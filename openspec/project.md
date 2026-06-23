# Project: Axis Movies

## Purpose
Axis Movies is a Radarr v3 API-compatible, Postgres-first, container-native movie
collection manager written in Go. It is the first app in the **Axis** suite
(`axis-movies`; future `axis-tv`, `axis-music`).

## Goals
- **Drop-in compatibility** with the Radarr v3 API so Prowlarr, Overseerr/Jellyseerr,
  dashboards, and mobile clients work unchanged.
- **Reliability**: Postgres-first storage and a durable, Postgres-backed job queue.
- **Performance & footprint**: a single static Go binary, small distroless image.
- **Mobile-first UI**: a SvelteKit PWA on the same public API.

## Non-goals
- Reimplementing indexer definitions (delegated to Prowlarr).
- SQLite support.
- Maintaining two parallel public APIs (v3 is the single contract; native
  extensions are additive and separately versioned).

## Tech stack
Go 1.26 · go-chi/chi v5 · PostgreSQL (pgx/v5) · golang-migrate · River (jobs) ·
sqlc (queries) · SvelteKit + Tailwind (UI) · distroless multi-arch images.

## Conventions
- Conventional commits. `gofmt` + `golangci-lint` clean. Table-driven tests.
- API shapes mirror Radarr v3; deviations are documented in JOURNEY.md.
- License GPLv3; the release parser is clean-room (no verbatim GPL copy).

## Workflow
Phased delivery tracked in `TASKS.md`; specs and change proposals live here in
`openspec/`. Each phase ends at a conformance/daily-driver gate.
