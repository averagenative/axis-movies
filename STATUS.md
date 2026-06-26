# Status — Axis Movies

**Phase:** 5 (download clients + grab) — core loop closed
**Updated:** 2026-06-26

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
- **Conformance gate PASSED**: a real Prowlarr (v2.4.0, on blacksky) successfully
  completes its "add as Radarr application" test against Axis. Required
  `GET /indexer/schema` (Torznab/Newznab, captured from real Radarr),
  `POST /indexer/test`, and an `X-Application-Version` header on all v3 responses.
- **TMDb metadata (Phase 2)**: `internal/tmdb` client (own API key via
  `AXIS_TMDB_API_KEY`); `GET /api/v3/movie/lookup?term=` (search) and
  `POST /api/v3/movie` (add by tmdbId — fetches metadata, persists, 409 on dup).
  Verified end-to-end against Postgres with a mock TMDb, and live against real
  TMDb. Without a key, lookup/add return 503.
- **Release parser (Phase 3)**: `internal/parser` clean-room parser extracting
  title/year/resolution/source/codec/proper/repack/group (+ best-effort
  audio/HDR/edition/language) across dotted-scene, YTS-bracket, anime front-group,
  and foreign formats. Validated 99/99 against a 5-agent-generated corpus and
  audited against 2342 real Radarr release names (1.2% anomalies, all benign).
  Not yet wired into the import path (that's Phase 4/6).
- **Indexer ingestion (Phase 4)**: DB-backed indexer CRUD
  (`POST/GET/PUT/DELETE /api/v3/indexer`). A real Prowlarr `fullSync` pushed 7
  movie-capable indexers into Axis end-to-end (verified live, then torn down).
- **Release search (Phase 4)**: `internal/torznab` (concurrent Torznab/Newznab
  feed search + XML parse) + `internal/quality` (resolution+source scoring) behind
  `GET /api/v3/release?movieId=` — searches all enabled indexers, parses each
  result via `internal/parser`, ranks best-first. **Verified live**: 452 real
  releases for Dune (2021) across 7 indexers, correctly parsed & ranked.
- **Download clients + grab (Phase 5)**: `internal/download` (qBittorrent WebUI v2
  + SABnzbd), download-client CRUD (`/api/v3/downloadclient`), and grab via
  `POST /api/v3/release` (picks a client by protocol, sends the magnet/nzb).
  **Verified live**: grab → real qBittorrent 5.x (torrent queued with category).
  The core loop — add → sync indexers → search → grab → download — now works
  end-to-end. Next: import pipeline (Phase 6), `/queue`, River jobs, real
  quality profiles.
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
Phase 2 remainder: TMDb response cache, image proxy, refresh job (the last needs
the Phase 4 job queue). Plus a live check against real TMDb once `AXIS_TMDB_API_KEY`
is set. Note: full indexer *sync* (Prowlarr pushing indexers via POST/PUT/DELETE
`/indexer`) is still Phase 4 — only the schema/test handshake is implemented so far.
