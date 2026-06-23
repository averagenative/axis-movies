// Package v3 implements the Radarr v3-compatible HTTP API.
//
// Compatibility note: ecosystem tools (Prowlarr, Overseerr/Jellyseerr, mobile
// clients like nzb360/LunaSea) identify the app via GET /system/status and
// string-match appName == "Radarr" and a minimum version. Axis therefore
// reports a Radarr-compatible appName/version by default (configurable) while
// surfacing its true identity in the axis* fields. This is the v1 read surface;
// write endpoints (grab/search/queue) land in Phase 5 (see TASKS.md).
package v3

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/version"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// compatRadarrVersion is the Radarr version Axis advertises for ecosystem
// minimum-version checks. Bump as compatibility coverage grows.
const compatRadarrVersion = "5.14.0.9383"

// Deps are the API's runtime dependencies.
type Deps struct {
	Config config.Config
	Log    *slog.Logger
	Pool   *pgxpool.Pool
}

// API holds the v3 handlers.
type API struct {
	deps Deps
}

// New constructs the API.
func New(deps Deps) *API { return &API{deps: deps} }

// Mount registers all v3 routes on the given router.
func (a *API) Mount(r chi.Router) {
	r.Get("/system/status", a.systemStatus)
	r.Get("/health", a.emptyArray)
	r.Get("/movie", a.emptyArray)
	r.Get("/rootfolder", a.emptyArray)
	r.Get("/tag", a.emptyArray)
	r.Get("/indexer", a.emptyArray)
	r.Get("/downloadclient", a.emptyArray)
	r.Get("/qualityprofile", a.qualityProfiles)
}

func (a *API) systemStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"appName":        a.deps.Config.CompatAppName,
		"instanceName":   a.deps.Config.InstanceName,
		"version":        compatRadarrVersion,
		"branch":         "master",
		"authentication": "apikey",
		"databaseType":   "postgreSQL",
		"isProduction":   true,
		"isDebug":        false,
		"isLinux":        true,
		"isDocker":       false,
		"runtimeName":    "go",
		"runtimeVersion": version.Version,
		"urlBase":        "",
		"isNetCore":      true,
		"mode":           "console",
		// Axis-native identity, ignored by Radarr clients.
		"axisApp":     "axis-movies",
		"axisVersion": version.Version,
		"axisCommit":  version.Commit,
	})
}

// qualityProfiles returns a minimal default profile so clients render a usable
// selector before the real profile engine lands in Phase 4.
func (a *API) qualityProfiles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]any{
		{
			"id":             1,
			"name":           "Any",
			"upgradeAllowed": true,
			"items":          []any{},
		},
	})
}

func (a *API) emptyArray(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
