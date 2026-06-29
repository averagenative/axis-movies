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

## 2026-06-26 — Phase 4 (slice 1): indexer ingestion

Indexers are stored in their own table (migration 0003) with the Radarr field
array (baseUrl, apiKey, categories, ...) kept verbatim as JSONB. Added the full
`/api/v3/indexer` CRUD (was previously a stub empty list): list, get, create,
update, delete, alongside the existing schema/test handlers. Request decoding is
lenient and derives protocol from implementation (Torznab→torrent, Newznab→usenet)
when omitted.

### Verified live against the real Prowlarr
Deployed Axis on blacksky's `nginx-proxy` network, added it to the real Prowlarr
as a Radarr application with `syncLevel: fullSync`, and triggered an
`ApplicationIndexerSync`. Prowlarr pushed **7 movie-capable indexers** into Axis
in ~1s (the 4 TV/book-only ones correctly skipped for a movie app), each with the
right protocol and per-indexer Prowlarr feed URL. Then cleaned up: deleted the
Prowlarr app and tore down the Axis containers, leaving the user's Prowlarr at its
original 3 apps.

### Next in Phase 4
The synced indexers carry Torznab/Newznab feed URLs; the next slice queries them
for releases, parses results via `internal/parser`, and scores them with the
decision engine — driven by River jobs (RSS sync + manual search).

## 2026-06-26 — Phase 4 (slice 2): release search pipeline

`internal/torznab` queries an indexer's Torznab/Newznab feed (as proxied by
Prowlarr) and parses the RSS/XML into items (title, size, seeders, download/magnet
URL). `internal/quality` maps a release's source+resolution to a Radarr-style
quality name and a numeric weight. `GET /api/v3/release?movieId=` loads the movie,
fans out concurrent searches across all enabled indexers, parses each result title
with `internal/parser`, scores it, and returns releases ranked best-first
(quality weight, then seeders). This is the first place the Phase-3 parser is
actually used in the request path.

### Verified live — 452 releases, real indexers
Against the user's real Prowlarr: synced 7 indexers, inserted Dune (2021), and
`GET /release` returned **452 real releases** across torrent + usenet, every one
parsed and ranked correctly (top result a Remux-2160p with 105 seeders). The
quality distribution (29 Remux-2160p, 141 Bluray-1080p, ...) confirmed the parser
on 452 live names with ~2% unknown — consistent with the offline audit. A bonus
real-world validation of Phase 3.

### Still simplified
Scoring is resolution+source weight only — no real quality profiles, cutoffs, or
custom formats yet. No grab action (that needs a download client, Phase 5) and no
RSS/background search (needs River). `t=search&q=Title Year` text query; could use
`t=movie&imdbid=` for precision later.

## 2026-06-26 — Phase 5: download clients + grab

`internal/download` defines a `Client` interface (Add + TestConnection) with two
implementations: qBittorrent (WebUI v2 — cookie login then torrents/add) and
SABnzbd (HTTP API mode=addurl). Download clients are stored like indexers (CRUD +
schema captured from real Radarr + test endpoint). `POST /api/v3/release` decodes
the chosen release, picks the highest-priority enabled client matching the
protocol, and sends the magnet/nzb.

### Live test caught a real version difference
Verified against a throwaway qBittorrent on blacksky. First attempt failed: my
client expected the classic `200 "Ok."` login response, but **qBittorrent 5.x
returns `204 No Content`** with the SID cookie. Fixed the login check to accept
204 (added a regression test). Re-ran: grab → the magnet landed in qBittorrent as
`queuedDL` with category `movies`. Used a fake-infohash magnet so nothing actually
downloads, and never touched the user's real Deluge/SABnzbd.

The core acquisition loop is now closed end-to-end: add movie → Prowlarr syncs
indexers → search → parse+score → grab → download client. What's left to be a
daily-driver: the import pipeline (Phase 6: detect completed download, hardlink/
rename into the library), a `/queue` view, River for RSS/automatic search, and
real quality profiles.

## 2026-06-29 — Phase 6: import pipeline (daily-driver gate)

`internal/importer` scans a completed download folder for the feature file
(largest video, skipping "sample"), and hardlinks it — falling back to a copy on
EXDEV (cross-filesystem) — into `<root>/<Title> (<year>)/<Title> (<year>)
[quality].ext`, sanitizing illegal path chars. The v3 import service parses the
file for quality, writes a `movie_file` (upsert) + a `history` row, and flips
`movie.has_file`. Triggered via `POST /api/v3/command` (DownloadedMoviesScan /
ManualImport, with movieId + path; other commands are accepted as no-ops so
clients don't error). `GET /history` (paged envelope) and `GET /moviefile` expose
the results.

### Verified with temp dirs, not the production library
The import is pure filesystem logic, so it's covered by a temp-dir + Postgres
integration test (feature picked over sample, hardlink confirmed via os.SameFile,
has_file set, moviefile + history reflect it) rather than a live run against the
user's real library — moving real files around in their collection is not worth
the risk when local tests fully exercise the logic.

### Daily-driver gate met
add movie → Prowlarr syncs indexers → search (parse+score) → grab → download
client → import (hardlink+rename into the library) now works end-to-end, each
stage verified. What remains is convenience/automation, not the core loop:
`/queue`, failed-download handling/blocklist, River for RSS+automatic search,
real quality profiles + custom formats, notifications (Phase 7), and the
SvelteKit PWA (Phase 8).
