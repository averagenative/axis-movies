package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/logging"
)

func testServer() http.Handler {
	return New(Deps{
		Config: config.Config{
			APIKey:        "secret",
			CompatAppName: "Radarr",
			InstanceName:  "Axis Movies",
		},
		Log: logging.New("error", "text"),
	}).Handler()
}

func TestPingIsUnauthenticated(t *testing.T) {
	rr := httptest.NewRecorder()
	testServer().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("ping: got %d, want 200", rr.Code)
	}
}

func TestSystemStatusRequiresAPIKey(t *testing.T) {
	rr := httptest.NewRecorder()
	testServer().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v3/system/status", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing key: got %d, want 401", rr.Code)
	}
}

func TestSystemStatusReportsCompatIdentity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v3/system/status", nil)
	req.Header.Set("X-Api-Key", "secret")
	rr := httptest.NewRecorder()
	testServer().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["appName"] != "Radarr" {
		t.Errorf("appName: got %v, want Radarr", body["appName"])
	}
	if body["axisApp"] != "axis-movies" {
		t.Errorf("axisApp: got %v, want axis-movies", body["axisApp"])
	}
}

func TestAPIKeyViaQueryParam(t *testing.T) {
	// Use a DB-free endpoint; this asserts the auth middleware accepts the
	// apikey query parameter. DB-backed endpoints are covered by the live
	// integration smoke test.
	req := httptest.NewRequest(http.MethodGet, "/api/v3/system/status?apikey=secret", nil)
	rr := httptest.NewRecorder()
	testServer().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("query apikey: got %d, want 200", rr.Code)
	}
}
