package v3

import (
	_ "embed"
	"net/http"
)

// indexerSchemaJSON is the Radarr v3 indexer schema for the generic Torznab and
// Newznab implementations, captured verbatim from Radarr so Prowlarr can build
// indexer definitions to push. Prowlarr's "add as Radarr application" test calls
// GET /api/v3/indexer/schema and fails without it.
//
//go:embed assets/indexer_schema.json
var indexerSchemaJSON []byte

// indexerSchema serves the Torznab/Newznab indexer schema.
func (a *API) indexerSchema(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(indexerSchemaJSON)
}

// indexerTest accepts the indexer definition Prowlarr posts during its
// "add as Radarr application" test. Phase 1 is a no-op accept (HTTP 200);
// real indexer connectivity testing lands with the decision engine in Phase 4.
func (a *API) indexerTest(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
