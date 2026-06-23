// Package server wires the HTTP router: middleware, health checks, and the
// Radarr v3-compatible API surface.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/averagenative/axis-movies/internal/api/v3"
	"github.com/averagenative/axis-movies/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are the runtime dependencies the server needs.
type Deps struct {
	Config config.Config
	Log    *slog.Logger
	Pool   *pgxpool.Pool
}

// Server is the HTTP application.
type Server struct {
	deps Deps
}

// New constructs a Server.
func New(deps Deps) *Server {
	return &Server{deps: deps}
}

// Handler builds the root http.Handler.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Unauthenticated liveness probe.
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	// Radarr v3-compatible API, guarded by API key.
	api := v3.New(v3.Deps{
		Config: s.deps.Config,
		Log:    s.deps.Log,
		Pool:   s.deps.Pool,
	})
	r.Route("/api/v3", func(r chi.Router) {
		r.Use(APIKey(s.deps.Config.APIKey))
		api.Mount(r)
	})

	return r
}
