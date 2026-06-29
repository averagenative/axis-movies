package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/db"
	"github.com/averagenative/axis-movies/internal/logging"
	"github.com/averagenative/axis-movies/internal/store"
	"github.com/go-chi/chi/v5"
)

func TestImportIntegration(t *testing.T) {
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
	for _, tbl := range []string{"history", "movie_file", "movie"} {
		_, _ = pool.Exec(ctx, "DELETE FROM "+tbl)
	}
	q := store.New(pool)

	// temp library root + a completed download folder with a feature + sample
	base := t.TempDir()
	root := filepath.Join(base, "library")
	dl := filepath.Join(base, "downloads", "Blade.Runner.2049.2017.1080p.BluRay.x264-GRP")
	if err := os.MkdirAll(dl, 0o755); err != nil {
		t.Fatal(err)
	}
	feature := filepath.Join(dl, "Blade.Runner.2049.2017.1080p.BluRay.x264-GRP.mkv")
	if err := os.WriteFile(feature, make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dl, "sample.mkv"), make([]byte, 10), 0o644)

	mv, err := q.CreateMovie(ctx, store.CreateMovieParams{
		TmdbID: 335984, Title: "Blade Runner 2049", Year: pgInt4(2017), Monitored: true,
		RootFolderPath: pgText(root), TitleSlug: pgText("blade-runner-2049-335984"),
		SortTitle: pgText("blade runner 2049"), Images: []byte("[]"), QualityProfileID: pgInt8(1),
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}

	api := New(Deps{Config: config.Config{}, Log: log, Pool: pool})
	r := chi.NewRouter()
	api.Mount(r)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	// trigger import
	rec := do(http.MethodPost, "/command",
		fmt.Sprintf(`{"name":"DownloadedMoviesScan","movieId":%d,"path":%q}`, mv.ID, dl))
	if rec.Code != http.StatusCreated {
		t.Fatalf("import command: got %d (%s), want 201", rec.Code, rec.Body.String())
	}

	// the file landed at the expected destination as a hard link to the source
	dest := filepath.Join(root, "Blade Runner 2049 (2017)", "Blade Runner 2049 (2017) [Bluray-1080p].mkv")
	di, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("imported file missing at %s: %v", dest, err)
	}
	si, _ := os.Stat(feature)
	if !os.SameFile(si, di) {
		t.Fatal("imported file is not a hardlink of the source")
	}

	// movie now has a file
	m, _ := q.GetMovie(ctx, mv.ID)
	if !m.HasFile {
		t.Fatal("movie.has_file not set after import")
	}

	// moviefile endpoint reflects it
	rec = do(http.MethodGet, "/moviefile?movieId="+strconv.FormatInt(mv.ID, 10), "")
	var files []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &files)
	if len(files) != 1 || !strings.Contains(files[0]["relativePath"].(string), "Blade Runner 2049 (2017)") {
		t.Fatalf("moviefile wrong: %+v", files)
	}

	// history recorded the import
	rec = do(http.MethodGet, "/history", "")
	var hist struct {
		Records []map[string]any `json:"records"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &hist)
	if len(hist.Records) == 0 || hist.Records[0]["eventType"] != "downloadFolderImported" {
		t.Fatalf("history wrong: %+v", hist.Records)
	}
}
