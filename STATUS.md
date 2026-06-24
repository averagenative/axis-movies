# Status — Axis Movies

**Phase:** 0 (Foundations) — in progress
**Updated:** 2026-06-23

## What works
- Go service builds, runs, and shuts down gracefully.
- Config from YAML + `AXIS_*` env (env wins); ephemeral API key auto-generated.
- Postgres connection pool (pgx/v5) + embedded golang-migrate migrations run on boot.
- Radarr v3-compatible **read** endpoints behind API-key auth:
  `/api/v3/system/status`, `/health`, `/movie`, `/rootfolder`, `/tag`,
  `/indexer`, `/downloadclient`, `/qualityprofile`. Unauthenticated `/ping`.
- `system/status` reports a Radarr-compatible identity (configurable `compat_app_name`).
- Docker (distroless static) + docker-compose (app + Postgres).
- Local quality gate via `make check` (gofmt, vet, race tests, build, golangci-lint)
  and an optional pre-push hook (`make install-hooks`). No GitHub Actions by design.

## What does NOT work yet
- No real movie data — endpoints return empty/stub payloads (no DB-backed reads).
- No metadata (TMDb), no release parsing, no decision engine, no download clients,
  no import pipeline, no notifications, no UI.
- Job queue is not yet wired (River lands in Phase 4).
- Not verified against a live Prowlarr/Overseerr yet (Phase 1 conformance gate).

## Known issues / notes
- `go.mod` pulls some heavy indirect deps via golang-migrate; trim later if needed.
- API surface is read-only; write endpoints (grab/search/queue) are Phase 5.

## Next
Phase 1: DB-backed movie/rootfolder/tag models + real v3 read endpoints, then the
**conformance gate** — get Prowlarr to add this as a "Radarr" application.
