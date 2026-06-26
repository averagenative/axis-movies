package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/db"
	"github.com/averagenative/axis-movies/internal/logging"
	"github.com/averagenative/axis-movies/internal/store"
	"github.com/go-chi/chi/v5"
)

const searchSampleXML = `<?xml version="1.0"?>
<rss xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel>
 <item><title>Blade Runner 2049 2017 2160p BluRay x265-GRP</title><size>50000000000</size>
  <enclosure url="http://dl/1.torrent" length="50000000000" type="application/x-bittorrent"/>
  <torznab:attr name="seeders" value="100"/><torznab:attr name="leechers" value="5"/></item>
 <item><title>Blade Runner 2049 2017 1080p WEB-DL x264-GRP</title><link>magnet:?xt=urn:btih:abc</link>
  <torznab:attr name="seeders" value="50"/><torznab:attr name="size" value="8000000000"/></item>
</channel></rss>`

func TestSearchReleasesIntegration(t *testing.T) {
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
	_, _ = pool.Exec(ctx, "DELETE FROM movie")
	_, _ = pool.Exec(ctx, "DELETE FROM indexer")
	q := store.New(pool)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(searchSampleXML))
	}))
	defer srv.Close()

	mv, err := q.CreateMovie(ctx, store.CreateMovieParams{
		TmdbID: 335984, Title: "Blade Runner 2049", Year: pgInt4(2017), Monitored: true,
		TitleSlug: pgText("blade-runner-2049-335984"), SortTitle: pgText("blade runner 2049"),
		Images: []byte("[]"), QualityProfileID: pgInt8(1),
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}
	fields := fmt.Sprintf(`[{"name":"baseUrl","value":%q},{"name":"apiPath","value":"/api"},{"name":"apiKey","value":"x"},{"name":"categories","value":[2000]}]`, srv.URL)
	if _, err := q.CreateIndexer(ctx, store.CreateIndexerParams{
		Name: "Mock", Implementation: "Torznab", ConfigContract: "TorznabSettings",
		Protocol: "torrent", Priority: 25, EnableRss: true, EnableAutomaticSearch: true,
		EnableInteractiveSearch: true, Fields: []byte(fields), Tags: []byte("[]"),
	}); err != nil {
		t.Fatalf("create indexer: %v", err)
	}

	api := New(Deps{Config: config.Config{}, Log: log, Pool: pool})
	r := chi.NewRouter()
	api.Mount(r)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/release?movieId="+strconv.FormatInt(mv.ID, 10), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("search: got %d (%s), want 200", rec.Code, rec.Body.String())
	}

	var rels []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &rels); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rels) != 2 {
		t.Fatalf("got %d releases, want 2", len(rels))
	}
	// best-first: the 2160p BluRay should rank above the 1080p WEB-DL
	q0 := rels[0]["quality"].(map[string]any)["quality"].(map[string]any)
	if q0["name"] != "Bluray-2160p" {
		t.Fatalf("first quality: %v, want Bluray-2160p", q0["name"])
	}
	if rels[0]["indexer"] != "Mock" || int(rels[0]["seeders"].(float64)) != 100 {
		t.Fatalf("first release wrong: %+v", rels[0])
	}
	q1 := rels[1]["quality"].(map[string]any)["quality"].(map[string]any)
	if q1["name"] != "WEBDL-1080p" {
		t.Fatalf("second quality: %v, want WEBDL-1080p", q1["name"])
	}
}
