# Axis Movies

A **Radarr v3 API-compatible**, **Postgres-first**, container-native movie
collection manager written in **Go** — the movies app of the **Axis** suite
(`axis-movies`, future `axis-tv`, `axis-music`).

> Status: **Phase 0 — Foundations.** Builds, runs, serves a read-only v3 API
> surface. Not yet usable as a Radarr replacement. See [TASKS.md](TASKS.md).

## Why this exists

[Radarr](https://github.com/Radarr/Radarr) is excellent but is a .NET app whose
default SQLite store is corruption-prone on container/network filesystems. Axis
is a ground-up Go reimplementation with three deliberate bets:

- **Drop-in compatibility.** Speak the Radarr v3 API so [Prowlarr](https://prowlarr.org/)
  manages indexers for us, and [Overseerr]/[Jellyseerr], dashboards, and mobile
  clients ([nzb360], [LunaSea]) work unchanged.
- **Postgres-first + durable jobs.** No SQLite. Schema via migrations; scheduled
  work via a Postgres-backed queue — no in-memory timers that lose state.
- **Mobile-first UI.** A SvelteKit PWA built on the same public API, embedded in
  a single static binary.

Indexers are **delegated to Prowlarr** (Torznab/Newznab), not reimplemented.

## Quick start (dev)

```
docker compose up --build
# API at http://localhost:7878  (api key: devkey0000000000000000000000000000)
curl -s localhost:7878/ping
curl -s -H "X-Api-Key: devkey0000000000000000000000000000" \
  localhost:7878/api/v3/system/status | jq
```

Run it directly against your own Postgres:

```
make run   # honours AXIS_* env vars / config.yaml
```

## Configuration

YAML file (via `AXIS_CONFIG`) overlaid by `AXIS_*` env vars (env wins). See
[`config.example.yaml`](config.example.yaml). Key vars: `AXIS_HTTP_ADDR`,
`AXIS_DATABASE_URL`, `AXIS_API_KEY`, `AXIS_LOG_FORMAT`.

## Tech stack

| Concern | Choice |
| --- | --- |
| Language | Go 1.26 |
| HTTP router | go-chi/chi v5 |
| Database | PostgreSQL (pgx/v5), Postgres-first |
| Migrations | golang-migrate (embedded SQL) |
| Jobs | Postgres-backed queue (River) — *Phase 4* |
| Type-safe queries | sqlc — *Phase 1* |
| Frontend | SvelteKit + Tailwind PWA — *Phase 8* |
| Packaging | Single static binary, distroless image, multi-arch |

## Layout

```
cmd/axis-movies      entrypoint
internal/config      config (yaml + AXIS_* env)
internal/server      chi router, middleware, API-key auth
internal/api/v3      Radarr v3-compatible handlers
internal/db          pgx pool + embedded migrations
internal/jobs        scheduler (River target) — Phase 4
internal/logging     slog setup
openspec/            specs and change proposals
web/                 SvelteKit PWA — Phase 8
```

## Roadmap

See [TASKS.md](TASKS.md) for the phased breakdown (sub-agent friendly) and
[openspec/](openspec/) for the spec. Short version: foundations → v3 read API →
metadata → release parser → decision engine → download clients → import pipeline
→ notifications → PWA → hardening.

## License

[GPLv3](LICENSE), matching the *arr ecosystem. The release-title parser is a
clean-room implementation; no GPL code is copied verbatim from Radarr.

[Overseerr]: https://overseerr.dev/
[Jellyseerr]: https://github.com/Fallenbagel/jellyseerr
[nzb360]: https://nzb360.com/
[LunaSea]: https://www.lunasea.app/
