// Package v3 implements the Radarr v3-compatible HTTP API.
//
// Compatibility note: ecosystem tools (Prowlarr, Overseerr/Jellyseerr, mobile
// clients like nzb360/LunaSea) identify the app via GET /system/status and
// string-match appName == "Radarr" and a minimum version. Axis therefore
// reports a Radarr-compatible appName/version by default (configurable) while
// surfacing its true identity in the axis* fields.
//
// Phase 1 provides the DB-backed read surface plus root-folder/tag mutation.
// Write endpoints that drive downloads (grab/search/queue) land in Phase 5.
package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/averagenative/axis-movies/internal/config"
	"github.com/averagenative/axis-movies/internal/store"
	"github.com/averagenative/axis-movies/internal/version"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	cfg config.Config
	log *slog.Logger
	q   *store.Queries
}

// New constructs the API.
func New(deps Deps) *API {
	return &API{
		cfg: deps.Config,
		log: deps.Log,
		q:   store.New(deps.Pool),
	}
}

// Mount registers all v3 routes on the given router.
func (a *API) Mount(r chi.Router) {
	r.Get("/system/status", a.systemStatus)
	r.Get("/health", a.emptyArray)

	r.Get("/movie", a.listMovies)
	r.Get("/movie/{id}", a.getMovie)

	r.Get("/rootfolder", a.listRootFolders)
	r.Get("/rootfolder/{id}", a.getRootFolder)
	r.Post("/rootfolder", a.createRootFolder)
	r.Delete("/rootfolder/{id}", a.deleteRootFolder)

	r.Get("/tag", a.listTags)
	r.Get("/tag/{id}", a.getTag)
	r.Post("/tag", a.createTag)
	r.Delete("/tag/{id}", a.deleteTag)

	r.Get("/qualityprofile", a.listQualityProfiles)
	r.Get("/qualityprofile/{id}", a.getQualityProfile)

	// Populated by Prowlarr (indexers) / configured later (download clients).
	r.Get("/indexer", a.emptyArray)
	r.Get("/downloadclient", a.emptyArray)
}

func (a *API) systemStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"appName":        a.cfg.CompatAppName,
		"instanceName":   a.cfg.InstanceName,
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

func (a *API) emptyArray(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

// --- shared helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"message": msg})
}

// idParam parses the {id} path parameter.
func idParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// decodeJSON decodes a JSON request body. It is deliberately lenient about
// unknown fields, since real Radarr clients post richer objects than we read.
func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func isNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
