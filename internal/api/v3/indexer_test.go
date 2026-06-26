package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/db"
	"github.com/averagenative/axis-movies/internal/logging"
	"github.com/go-chi/chi/v5"
)

// TestIndexerCRUDIntegration exercises the indexer endpoints Prowlarr drives
// during a sync (create/list/get/update/delete) against a real Postgres.
func TestIndexerCRUDIntegration(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "DELETE FROM indexer"); err != nil {
		t.Fatalf("clean: %v", err)
	}

	api := New(Deps{Config: config.Config{}, Log: log, Pool: pool})
	r := chi.NewRouter()
	api.Mount(r)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	// create — a Torznab indexer as Prowlarr would push it
	payload := `{"name":"1337x (Prowlarr)","implementation":"Torznab","implementationName":"Torznab","configContract":"TorznabSettings","fields":[{"name":"baseUrl","value":"http://prowlarr:9696/1/api"},{"name":"apiKey","value":"abc"},{"name":"categories","value":[2000,2010]}],"tags":[]}`
	rec := do(http.MethodPost, "/indexer", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d (%s), want 201", rec.Code, rec.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created["protocol"] != "torrent" || created["supportsRss"] != true {
		t.Fatalf("unexpected created indexer: %+v", created)
	}
	id := int64(created["id"].(float64))

	// fields round-trip preserved
	fields, _ := json.Marshal(created["fields"])
	if !strings.Contains(string(fields), "baseUrl") || !strings.Contains(string(fields), "2010") {
		t.Fatalf("fields not preserved: %s", fields)
	}

	// list
	rec = do(http.MethodGet, "/indexer", "")
	var list []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("list: got %d, want 1", len(list))
	}

	// get by id
	rec = do(http.MethodGet, "/indexer/"+itoa(id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get: got %d, want 200", rec.Code)
	}

	// update priority
	rec = do(http.MethodPut, "/indexer/"+itoa(id),
		`{"name":"1337x (Prowlarr)","implementation":"Torznab","configContract":"TorznabSettings","priority":40,"fields":[],"tags":[]}`)
	var updated map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if int(updated["priority"].(float64)) != 40 {
		t.Fatalf("update priority: got %v, want 40", updated["priority"])
	}

	// delete -> list empty
	rec = do(http.MethodDelete, "/indexer/"+itoa(id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: got %d, want 200", rec.Code)
	}
	rec = do(http.MethodGet, "/indexer", "")
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("after delete: got %d, want 0", len(list))
	}
}

func itoa(i int64) string {
	return strings.TrimSpace(strconv.FormatInt(i, 10))
}
