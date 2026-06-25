package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/db"
	"github.com/averagenative/axis-movies/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func mockTMDb(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			_, _ = w.Write([]byte(`{"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","overview":"A hacker.","poster_path":"/p.jpg","backdrop_path":"/b.jpg"}]}`))
		case r.URL.Path == "/movie/603":
			_, _ = w.Write([]byte(`{"id":603,"imdb_id":"tt0133093","title":"The Matrix","release_date":"1999-03-30","overview":"A hacker.","runtime":136,"status":"Released","poster_path":"/p.jpg","backdrop_path":"/b.jpg"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func routerFor(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	api := New(Deps{Config: cfg, Log: logging.New("error", "text"), Pool: pool})
	r := chi.NewRouter()
	api.Mount(r)
	return r
}

func TestLookupMovie(t *testing.T) {
	srv := mockTMDb(t)
	defer srv.Close()
	cfg := config.Config{TMDBAPIKey: "k", TMDBBaseURL: srv.URL, TMDBImageBaseURL: "https://img"}
	r := routerFor(cfg, nil)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/movie/lookup?term=matrix", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("lookup: got %d, want 200", rec.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || int(got[0]["tmdbId"].(float64)) != 603 || got[0]["title"] != "The Matrix" {
		t.Fatalf("unexpected lookup result: %+v", got)
	}
}

func TestLookupNotConfigured(t *testing.T) {
	r := routerFor(config.Config{}, nil) // no TMDb key
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/movie/lookup?term=x", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("lookup without key: got %d, want 503", rec.Code)
	}
}

// TestAddMovieIntegration exercises the full add path against a real Postgres
// (set AXIS_TEST_DATABASE_URL) with a mock TMDb — no real API key needed.
func TestAddMovieIntegration(t *testing.T) {
	url := os.Getenv("AXIS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set AXIS_TEST_DATABASE_URL to run the DB integration test")
	}
	ctx := context.Background()
	log := logging.New("error", "text")
	if err := db.Migrate(url, log); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "DELETE FROM movie"); err != nil {
		t.Fatalf("clean: %v", err)
	}

	srv := mockTMDb(t)
	defer srv.Close()
	cfg := config.Config{TMDBAPIKey: "k", TMDBBaseURL: srv.URL, TMDBImageBaseURL: "https://img"}
	api := New(Deps{Config: cfg, Log: log, Pool: pool})
	r := chi.NewRouter()
	api.Mount(r)

	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		r.ServeHTTP(rec, req)
		return rec
	}

	// add
	rec := do(http.MethodPost, "/movie", `{"tmdbId":603,"rootFolderPath":"/movies"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add: got %d (%s), want 201", rec.Code, rec.Body.String())
	}
	var m map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	if int(m["tmdbId"].(float64)) != 603 || m["title"] != "The Matrix" || int(m["qualityProfileId"].(float64)) != 1 {
		t.Fatalf("unexpected created movie: %+v", m)
	}
	if m["path"] != "/movies/The Matrix (1999)" {
		t.Fatalf("unexpected path: %v", m["path"])
	}

	// list shows it
	rec = do(http.MethodGet, "/movie", "")
	var list []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("list: got %d, want 1", len(list))
	}

	// duplicate add -> 409
	rec = do(http.MethodPost, "/movie", `{"tmdbId":603,"rootFolderPath":"/movies"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate add: got %d, want 409", rec.Code)
	}
}
