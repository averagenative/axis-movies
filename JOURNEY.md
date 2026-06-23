# Journey — design decisions & rationale

Chronological record of why Axis Movies is built the way it is.

## 2026-06-23 — Project bootstrap

### Decision: Radarr v3 API compatibility as the single public contract
Rather than invent a new API, Axis implements the Radarr v3 API. This makes
Prowlarr push indexers to us for free and lets Overseerr/Jellyseerr, dashboards,
and mobile clients (nzb360, LunaSea) work unchanged. **One contract, two
consumers** (ecosystem + our own PWA) — we explicitly avoid maintaining two
parallel APIs. Native-only extensions, when needed, live under a separate
versioned prefix.

### Decision: report `appName: "Radarr"` by default
Ecosystem tools string-match the app type via `/system/status`. To be a true
drop-in we advertise a Radarr-compatible `appName` and version, while exposing
our real identity in `axisApp`/`axisVersion` fields. Configurable via
`compat_app_name` for users who'd rather not masquerade.

### Decision: Postgres-first, no SQLite
The corruption pain that motivates this project is SQLite-on-container/network-FS.
Radarr *added* Postgres support (v4.1+), so Postgres alone isn't novel — our edge
is Postgres-*first* + a durable job queue + API-compat + a real PWA, combined.
Dropping SQLite keeps one SQL dialect and a simpler, more reliable core.

### Decision: delegate indexers to Prowlarr
Maintaining hundreds of tracker definitions is the largest ongoing cost in this
space. Being v3-compatible means Prowlarr manages indexers and pushes Torznab/
Newznab config to us. We consume feeds; we don't curate definitions.

### Decision: Go, chi, pgx, golang-migrate; River + sqlc deferred
Go gives a single static binary, great I/O concurrency, and an easy contributor
on-ramp. Phase 0 wires chi + pgx + embedded migrations and stays dependency-light.
River (Postgres-backed jobs) and sqlc (type-safe queries) are deliberately
deferred to the phases that first need them, to keep the foundation compiling and
honest rather than stubbing large frameworks up front.

### Prior art reviewed
`Kellerman81/go_media_downloader` proves a Go *arr-like manager is feasible but
uses its own API (no Prowlarr/Overseerr/mobile compatibility), so it validates
feasibility without occupying our niche. Good reference for Go download-client and
Torznab handling; we do not fork it. `bobarr` (TS, monolithic) and the Python
TV tools (Medusa, SickGear) are different philosophies. The API-compat + Postgres-
first + PWA combination is unclaimed in the non-.NET space.
