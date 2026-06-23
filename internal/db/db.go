// Package db owns the Postgres connection pool and schema migrations.
//
// Axis is Postgres-first: there is no SQLite fallback. Schema is managed with
// golang-migrate using SQL files embedded from internal/db/migrations.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers the "pgx5" scheme
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect opens and verifies a pgx connection pool.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// Migrate applies all pending up migrations against the target database.
func Migrate(url string, log *slog.Logger) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, normalizeURL(url))
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	v, dirty, verr := m.Version()
	if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
		return fmt.Errorf("migrate version: %w", verr)
	}
	log.Info("migrations applied", "schema_version", v, "dirty", dirty)
	return nil
}

// normalizeURL rewrites a postgres:// connection string to the pgx5:// scheme
// expected by the golang-migrate pgx/v5 database driver.
func normalizeURL(url string) string {
	if rest, ok := strings.CutPrefix(url, "postgres://"); ok {
		return "pgx5://" + rest
	}
	if rest, ok := strings.CutPrefix(url, "postgresql://"); ok {
		return "pgx5://" + rest
	}
	return url
}
