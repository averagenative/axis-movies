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

## 2026-06-25 — Conformance gate passed against a real Prowlarr

Tested against the user's real Prowlarr v2.4.0 (linuxserver, on `blacksky`),
running Axis + Postgres as throwaway containers on Prowlarr's `nginx-proxy`
docker network so Prowlarr could reach it by name. Prowlarr's "add as Radarr
application" test exercised more than system/status; iterating on each failure
revealed the true minimum surface:

1. `GET /api/v3/indexer/schema` — Prowlarr builds indexer definitions from it.
   Captured the generic **Torznab + Newznab** schema verbatim from the real Radarr
   on the same box and embed it (`assets/indexer_schema.json`); served as-is.
2. `POST /api/v3/indexer/test` — Prowlarr validates the indexer it would push.
   Phase 1 returns a no-op 200 (real indexer connectivity testing is Phase 4).
3. **`X-Application-Version` response header** — the actual blocker. Prowlarr's
   `TestConnection` reads the app version from this header on the indexer/test
   response, *not* from system/status JSON. Missing header → "Failed to fetch
   Radarr version". Radarr sets it on every response, so Axis now does too (v3
   middleware). Confirmed by reading Prowlarr's `RadarrV3Proxy.cs` source.

Result: the application test returns **HTTP 200**. Gate cleared.

Aside, on-thesis: during testing Prowlarr logged
`SQLiteException: database disk image is malformed` from its own DB — the exact
SQLite-corruption failure mode that motivates Axis being Postgres-first, observed
live in a production *arr stack.

Scope note: only the schema/test handshake the *application test* needs is built.
Actual indexer **sync** (Prowlarr POST/GET/PUT/DELETE `/api/v3/indexer`) remains
Phase 4.

## 2026-06-25 — Phase 2: TMDb metadata (lookup + add)

`internal/tmdb` is a thin TMDb v3 client (search, movie details, image URLs).
Axis brings its **own** API key (`AXIS_TMDB_API_KEY`) — it cannot use Radarr's
`api.radarr.video` proxy. Base URLs are configurable purely so tests can point at
a mock server.

Endpoints: `GET /api/v3/movie/lookup?term=` (search → Radarr lookup shape) and
`POST /api/v3/movie` (add by `tmdbId`: re-fetch details from TMDb rather than
trusting the client, generate a `title-slug-<tmdbId>`, derive
`<root>/<Title> (<year>)` path, default to the seeded quality profile, persist).
Duplicates → 409; no key → 503.

### Decision: test with a mock TMDb, not a real key
The add path is verified by `TestAddMovieIntegration`, which runs against a real
Postgres (gated on `AXIS_TEST_DATABASE_URL`) with an `httptest` mock TMDb server.
This gives deterministic, key-free end-to-end coverage of search → add → list →
duplicate. A real-key smoke test is left as a follow-up.

### Deferred within Phase 2
TMDb response caching (currently hits TMDb live), local image proxy (we serve the
TMDb CDN `remoteUrl` directly), and the refresh-metadata job (needs the Phase 4
job queue).

## 2026-06-26 — Phase 3: release-name parser

`internal/parser` is a clean-room, token-based movie release-name parser. Design:
extract the group first (anime front-`[bracket]`, YTS/YIFY tail bracket, or scene
`-GROUP` with a false-dash guard), normalize separators to spaces, detect quality
attributes with keyword regexes, then split title/year by finding the first
*hard* quality anchor and taking the last year before it.

### Decision: corpus built by a 5-agent fan-out, parser by me
The parser (interdependent) was written solo; the **ground-truth corpus** was
generated by 5 parallel sub-agents covering distinct slices (UHD/REMUX, streaming
WEB, older HDTV/DVDRip, edge cases, YTS/anime/foreign). Independent generation
avoids the circular trap of testing a parser against its author's own assumptions.
One agent initially returned TV episodes — redirected to movies via SendMessage.
A few agent labels were "prettified" with unrecoverable punctuation (`Spider-Man:
No Way Home`), so the corpus test compares titles with punctuation/case stripped
and everything else exactly. Result: 99/99.

### Real-collection audit drove the hard bugs out
`TestAuditCollection` (gated on `AXIS_COLLECTION_FILE`) parsed 2342 real release
names pulled from the user's actual Radarr. The synthetic corpus passed 100% but
the real data exposed three bugs the corpus missed:
1. edition/language tokens *before* the year ("Extended Edition 2001") were used
   as title anchors, hiding the year → excluded them from anchors + strip them
   from the title tail.
2. bare common words that are also quality tokens — **"Web"** (Madame Web,
   Charlotte's Web) and **"Opus"** (the 2025 film) — matched as anchors → removed
   from the anchor set.
3. hyphen-glued years ("Valkyrie-2008-1080p") → split in normalization.
Final: 2342 names, 1.2% anomalies, all benign (names genuinely lacking quality
tags, or truly yearless). Lesson: synthetic corpus for breadth, real data for the
long tail.
