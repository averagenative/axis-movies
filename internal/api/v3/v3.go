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
	"github.com/averagenative/axis-movies/internal/tmdb"
	"github.com/averagenative/axis-movies/internal/torznab"
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
	cfg  config.Config
	log  *slog.Logger
	q    *store.Queries
	tmdb *tmdb.Client
	tz   *torznab.Client
}

// New constructs the API.
func New(deps Deps) *API {
	return &API{
		cfg:  deps.Config,
		log:  deps.Log,
		q:    store.New(deps.Pool),
		tmdb: tmdb.New(deps.Config.TMDBAPIKey, deps.Config.TMDBBaseURL, deps.Config.TMDBImageBaseURL, nil),
		tz:   torznab.New(nil),
	}
}

// Mount registers all v3 routes on the given router.
func (a *API) Mount(r chi.Router) {
	// Radarr sets X-Application-Version on every response; Prowlarr reads it
	// from POST /indexer/test to learn the app version. Without it, Prowlarr's
	// application test fails with "Failed to fetch Radarr version".
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Application-Version", compatRadarrVersion)
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/system/status", a.systemStatus)
	r.Get("/health", a.emptyArray)

	r.Get("/movie/lookup", a.lookupMovie)
	r.Get("/movie", a.listMovies)
	r.Post("/movie", a.addMovie)
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

	// Indexers — managed (pushed) by Prowlarr.
	r.Get("/indexer/schema", a.indexerSchema)
	r.Post("/indexer/test", a.indexerTest)
	r.Post("/indexer/testall", a.indexerTest)
	r.Get("/indexer", a.listIndexers)
	r.Post("/indexer", a.createIndexer)
	r.Get("/indexer/{id}", a.getIndexer)
	r.Put("/indexer/{id}", a.updateIndexer)
	r.Delete("/indexer/{id}", a.deleteIndexer)

	// Download clients (qBittorrent / SABnzbd).
	r.Get("/downloadclient/schema", a.downloadClientSchema)
	r.Post("/downloadclient/test", a.downloadClientTest)
	r.Get("/downloadclient", a.listDownloadClients)
	r.Post("/downloadclient", a.createDownloadClient)
	r.Get("/downloadclient/{id}", a.getDownloadClient)
	r.Put("/downloadclient/{id}", a.updateDownloadClient)
	r.Delete("/downloadclient/{id}", a.deleteDownloadClient)

	// Release search (GET) and grab (POST) across the synced indexers.
	r.Get("/release", a.searchReleases)
	r.Post("/release", a.grabRelease)
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
