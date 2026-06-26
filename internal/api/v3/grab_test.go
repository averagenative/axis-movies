package v3

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/db"
	"github.com/averagenative/axis-movies/internal/logging"
	"github.com/averagenative/axis-movies/internal/store"
	"github.com/go-chi/chi/v5"
)

func TestGrabIntegration(t *testing.T) {
	dbURL := os.Getenv("AXIS_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("set AXIS_TEST_DATABASE_URL to run the DB integration test")
	}
	ctx := context.Background()
	log := logging.New("error", "text")
	if err := db.Migrate(dbURL, log); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	_, _ = pool.Exec(ctx, "DELETE FROM download_client")
	q := store.New(pool)

	// mock qBittorrent
	var added string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/add":
			_ = r.ParseForm()
			added = r.FormValue("urls")
			_, _ = w.Write([]byte("Ok."))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	fields := fmt.Sprintf(`[{"name":"host","value":%q},{"name":"port","value":%s},{"name":"username","value":"admin"},{"name":"password","value":"pw"},{"name":"movieCategory","value":"movies"}]`, host, portStr)
	if _, err := q.CreateDownloadClient(ctx, store.CreateDownloadClientParams{
		Name: "qbit", Implementation: "QBittorrent", ConfigContract: "QBittorrentSettings",
		Protocol: "torrent", Priority: 1, Enable: true, Fields: []byte(fields), Tags: []byte("[]"),
	}); err != nil {
		t.Fatalf("create download client: %v", err)
	}

	api := New(Deps{Config: config.Config{}, Log: log, Pool: pool})
	r := chi.NewRouter()
	api.Mount(r)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	// grab a torrent -> qbittorrent receives the magnet
	rec := do(http.MethodPost, "/release",
		`{"guid":"x","title":"Dune 2021 2160p BluRay x265-GRP","magnetUrl":"magnet:?xt=urn:btih:abc","protocol":"torrent","movieId":1}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("grab: got %d (%s), want 201", rec.Code, rec.Body.String())
	}
	if added != "magnet:?xt=urn:btih:abc" {
		t.Fatalf("qbittorrent did not receive magnet, got %q", added)
	}

	// no usenet client configured -> 409
	rec = do(http.MethodPost, "/release",
		`{"guid":"y","title":"x","downloadUrl":"http://x/1.nzb","protocol":"usenet"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("usenet grab without client: got %d, want 409", rec.Code)
	}
}
