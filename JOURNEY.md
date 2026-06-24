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

## 2026-06-24 — Phase 1: DB-backed read surface

### Decision: sqlc for the query layer
Adopted sqlc (pgx/v5 mode) generating `internal/store` from SQL in
`internal/db/queries`, with the schema sourced directly from the golang-migrate
files. Generated code is committed so building never requires the sqlc binary —
only regenerating does. Nullable columns surface as pgtype.* and are mapped to
plain JSON values in a small `mapping.go` layer. Trade-off accepted: a little
pgtype verbosity in exchange for compile-time-checked queries as the surface grows.

### Decision: lenient request decoding
v3 write endpoints (rootfolder/tag create) decode JSON without
DisallowUnknownFields. Real Radarr clients post richer objects than we read;
rejecting unknown fields would break compatibility. Postel's law for an API-compat
layer.

### Decision: defer history/blocklist tables
The Phase 1 task listed them, but no endpoint exposes them yet. Creating unused
tables now is premature; they land with the grab/import phases that need them.

### Scope landed
Migration 0002 (quality_profile, tag, expanded movie, seeded "Any" profile) +
DB-backed movie (read), rootfolder (CRUD), tag (CRUD), qualityprofile (read).
Verified end-to-end against Postgres. The conformance gate (live Prowlarr adds
Axis as "Radarr") now only awaits a real Prowlarr instance — the endpoints it
checks are all serving real data.
