# Status — Axis Movies

**Phase:** 1 (v3 API read surface) — in progress
**Updated:** 2026-06-24

## What works
- Go service builds, runs, and shuts down gracefully.
- Config from YAML + `AXIS_*` env (env wins); ephemeral API key auto-generated.
- Postgres connection pool (pgx/v5) + embedded golang-migrate migrations on boot
  (schema v2: quality_profile, tag, expanded movie; default "Any" profile seeded).
- **sqlc**-generated type-safe store layer (`internal/store`) over pgx.
- DB-backed Radarr v3 endpoints behind API-key auth:
  - `GET /movie`, `GET /movie/{id}`
  - `GET/POST/DELETE /rootfolder` (+ `GET /rootfolder/{id}`)
  - `GET/POST/DELETE /tag` (+ `GET /tag/{id}`)
  - `GET /qualityprofile` (+ `/{id}`) — returns the seeded default
  - `GET /system/status` (Radarr-compatible identity), `/health`, `/indexer`,
    `/downloadclient`. Unauthenticated `/ping`.
- Verified end-to-end against Postgres: CRUD round-trips, 409 on duplicate,
  404 on missing, delete works.
- Docker (distroless static) + docker-compose (app + Postgres).
- Local `make check` gate (gofmt, vet, race tests, build, golangci-lint v2 — 0 issues)
  + optional pre-push hook. No GitHub Actions by design.

## What does NOT work yet
- Movies are read-only and the table starts empty — adding movies needs TMDb
  metadata (Phase 2). No release parsing, decision engine, download clients,
  import pipeline, notifications, or UI yet.
- Job queue not wired (River lands in Phase 4).
- `/indexer` and `/downloadclient` return empty arrays (populated in Phases 4–5).

## Known issues / notes
- `go.mod` pulls some heavy indirect deps via golang-migrate; trim later if needed.
- No in-repo live-Postgres integration test yet (verified manually via podman);
  testcontainers test is an open Phase 0 task.

## Next
**Conformance gate:** point a real Prowlarr at this instance and confirm it adds
Axis as a "Radarr" application (system/status + qualityprofile/rootfolder/tag are
all in place). Then Phase 2 — TMDb metadata + movie add/lookup.
