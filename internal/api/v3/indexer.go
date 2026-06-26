package v3

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/averagenative/axis-movies/internal/store"
)

// indexerSchemaJSON is the Radarr v3 indexer schema for the generic Torznab and
// Newznab implementations, captured verbatim from Radarr so Prowlarr can build
// indexer definitions to push.
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
// "add as Radarr application" test. Real indexer connectivity testing (actually
// querying the Torznab feed) lands with the search pipeline.
func (a *API) indexerTest(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// indexerRequest is the Radarr v3 indexer resource as posted by Prowlarr.
type indexerRequest struct {
	Name                    string          `json:"name"`
	Implementation          string          `json:"implementation"`
	ConfigContract          string          `json:"configContract"`
	Protocol                string          `json:"protocol"`
	Priority                int32           `json:"priority"`
	EnableRss               *bool           `json:"enableRss"`
	EnableAutomaticSearch   *bool           `json:"enableAutomaticSearch"`
	EnableInteractiveSearch *bool           `json:"enableInteractiveSearch"`
	Fields                  json.RawMessage `json:"fields"`
	Tags                    json.RawMessage `json:"tags"`
}

func (a *API) listIndexers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.q.ListIndexers(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, ix := range rows {
		out = append(out, indexerJSON(ix))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getIndexer(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ix, err := a.q.GetIndexer(r.Context(), id)
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "indexer not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, indexerJSON(ix))
	}
}

func (a *API) createIndexer(w http.ResponseWriter, r *http.Request) {
	var b indexerRequest
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if b.Name == "" || b.Implementation == "" {
		writeError(w, http.StatusBadRequest, "name and implementation are required")
		return
	}
	ix, err := a.q.CreateIndexer(r.Context(), store.CreateIndexerParams{
		Name:                    b.Name,
		Implementation:          b.Implementation,
		ConfigContract:          b.ConfigContract,
		Protocol:                protocolFor(b.Implementation, b.Protocol),
		Priority:                priorityOr(b.Priority),
		EnableRss:               boolOr(b.EnableRss, true),
		EnableAutomaticSearch:   boolOr(b.EnableAutomaticSearch, true),
		EnableInteractiveSearch: boolOr(b.EnableInteractiveSearch, true),
		Fields:                  rawOrEmpty(b.Fields),
		Tags:                    rawOrEmpty(b.Tags),
	})
	if err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, indexerJSON(ix))
}

func (a *API) updateIndexer(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var b indexerRequest
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ix, err := a.q.UpdateIndexer(r.Context(), store.UpdateIndexerParams{
		ID:                      id,
		Name:                    b.Name,
		Implementation:          b.Implementation,
		ConfigContract:          b.ConfigContract,
		Protocol:                protocolFor(b.Implementation, b.Protocol),
		Priority:                priorityOr(b.Priority),
		EnableRss:               boolOr(b.EnableRss, true),
		EnableAutomaticSearch:   boolOr(b.EnableAutomaticSearch, true),
		EnableInteractiveSearch: boolOr(b.EnableInteractiveSearch, true),
		Fields:                  rawOrEmpty(b.Fields),
		Tags:                    rawOrEmpty(b.Tags),
	})
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "indexer not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, indexerJSON(ix))
	}
}

func (a *API) deleteIndexer(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := a.q.DeleteIndexer(r.Context(), id); err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

func indexerJSON(ix store.Indexer) map[string]any {
	return map[string]any{
		"id":                      ix.ID,
		"name":                    ix.Name,
		"implementation":          ix.Implementation,
		"implementationName":      ix.Implementation,
		"configContract":          ix.ConfigContract,
		"protocol":                ix.Protocol,
		"priority":                ix.Priority,
		"enableRss":               ix.EnableRss,
		"enableAutomaticSearch":   ix.EnableAutomaticSearch,
		"enableInteractiveSearch": ix.EnableInteractiveSearch,
		"supportsRss":             true,
		"supportsSearch":          true,
		"fields":                  rawJSON(ix.Fields),
		"tags":                    rawJSON(ix.Tags),
	}
}

func protocolFor(impl, given string) string {
	if given != "" {
		return given
	}
	if strings.EqualFold(impl, "Newznab") {
		return "usenet"
	}
	return "torrent"
}

func priorityOr(p int32) int32 {
	if p == 0 {
		return 25
	}
	return p
}

func boolOr(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

func rawOrEmpty(r json.RawMessage) []byte {
	if len(r) == 0 {
		return []byte("[]")
	}
	return r
}
