# Tasks — Axis Movies

Phased roadmap. Each task is sized to be **independently ownable** by a sub-agent
in its own worktree. Conformance gates (✅-marked goals) are the "is this phase
real" checkpoints. `[x]` done, `[ ]` open, `[~]` in progress.

Legend: **[P]** parallelizable with siblings · **[S]** sequential dependency.

---

## Phase 0 — Foundations  `[~]`
- [x] Go module, repo, GPLv3, README
- [x] Config (YAML + `AXIS_*` env), structured logging (slog)
- [x] Postgres pool (pgx/v5) + embedded golang-migrate migrations on boot
- [x] chi router, middleware, API-key auth, `/ping`
- [x] Radarr v3 read stubs (system/status, health, movie, rootfolder, tag, indexer, downloadclient, qualityprofile)
- [x] Dockerfile (distroless static) + docker-compose (app + Postgres)
- [x] Local quality gate: `make check` + optional pre-push hook (no GitHub Actions, by design — avoids any CI cost)
- [x] [P] Add `sqlc` config + generate typed query layer scaffolding (`internal/store`)
- [ ] [P] Multi-arch release build (local/manual goreleaser or buildx amd64+arm64) + image publish
- [ ] [P] `/system/status` integration test against a live Postgres (testcontainers)

## Phase 1 — v3 API: read surface (real data)  `[~]`
- [x] [S] Domain models + migrations: movie (expanded), root_folder, quality_profile, tag (history/blocklist deferred to grab/import phases)
- [x] [P] `GET/POST/DELETE /api/v3/rootfolder` (DB-backed, + `/{id}`)
- [x] [P] `GET/POST/DELETE /api/v3/tag` (DB-backed, + `/{id}`)
- [x] [P] `GET /api/v3/movie` + `GET /api/v3/movie/{id}` (DB-backed)
- [x] [P] `GET /api/v3/qualityprofile` (+ `/{id}`); default "Any" seeded
- [ ] [S] Pagination/sort envelope matching Radarr (`/api/v3/movie/lookup` later)
- [x] [S] `GET /indexer/schema` (Torznab/Newznab) + `POST /indexer/test` + `X-Application-Version` header — required by the Prowlarr application test
- [x] ✅ **Conformance gate PASSED:** real Prowlarr v2.4.0 (blacksky) completes its "add as Radarr application" test against Axis (2026-06-25)
- [ ] Overseerr & nzb360 connect read-only (not yet verified)

## Phase 2 — Metadata  `[~]`
- [x] [S] TMDb client (own API key, `internal/tmdb`) — search + details + image URLs
- [x] [P] `GET /api/v3/movie/lookup?term=` (TMDb-backed search)
- [x] [P] `POST /api/v3/movie` (add by tmdbId) — fetches metadata, persists, 409 on dup
- [ ] [P] Postgres response cache for TMDb (currently hits TMDb directly)
- [ ] [P] Image proxy/cache (currently serves TMDb CDN `remoteUrl` directly)
- [ ] [P] Refresh-movie metadata job (needs the job queue — Phase 4)
- [ ] [P] Live verification against real TMDb (needs a real `AXIS_TMDB_API_KEY`)

## Phase 3 — Release parser (crown jewels)  `[~]`
- [x] [S] Clean-room parser (`internal/parser`): title, year, resolution, source, codec, proper/repack, group; best-effort audio/HDR/edition/language
- [x] [P] **Test corpus** (`testdata/corpus.json`, 99 entries from a 5-agent fan-out: UHD/BluRay/REMUX, streaming WEB, older HDTV/DVDRip, edge cases, YTS/anime/foreign)
- [x] [P] Property/regression test vs corpus (lenient title, exact fields) — 99/99
- [x] [P] Scene/anime/repack/foreign edge-case coverage (YTS brackets, anime front-group, title-with-year, hyphen-years)
- [x] [P] Real-collection audit tool (`TestAuditCollection`, gated on `AXIS_COLLECTION_FILE`) — audited 2342 real Radarr names, 1.2% anomalies, all correct/informational
- [ ] [P] Wire the parser into the import/decision path (Phase 4/6)

## Phase 4 — Decision engine + indexer ingestion  `[~]`
- [x] [S] Indexer persistence + CRUD (`POST/GET/PUT/DELETE /api/v3/indexer`) — Prowlarr `fullSync` pushes its indexers into Axis. **Verified live**: real Prowlarr synced 7 movie-capable indexers into Axis end-to-end.
- [ ] [S] Wire **River** (Postgres-backed job queue); migrate scheduler onto it
- [ ] [S] Torznab/Newznab feed consumer (query the synced indexers for releases; parse via `internal/parser`)
- [ ] [P] Quality profile evaluation + custom-format scoring
- [ ] [P] Upgrade-until / min-max size / age / preferred-words logic
- [ ] [P] RSS sync job + manual search command

## Phase 5 — Download clients + grab flow  `[ ]`
- [ ] [S] Download-client interface
- [ ] [P] qBittorrent client
- [ ] [P] SABnzbd client
- [ ] [P] v3 write endpoints: `/release` (grab), `/command` (search), `/queue`
- [ ] [ ] Later: Transmission, Deluge, NZBGet, rTorrent

## Phase 6 — Import pipeline  `[ ]`
- [ ] [S] Completed-download handling + parse + match to movie
- [ ] [P] Hardlink/move with naming-token rename engine
- [ ] [P] Import history + manual import endpoint
- [ ] [P] Failed-download handling / blocklist
- [ ] ✅ **Daily-driver gate:** end-to-end add → search → grab → import works for the maintainer

## Phase 7 — Notifications & connect  `[ ]`
- [ ] [P] Webhook + Discord
- [ ] [P] Jellyfin/Plex/Emby library refresh on import
- [ ] [P] `/notification` v3 endpoints

## Phase 8 — SvelteKit PWA  `[ ]`
- [ ] [S] SvelteKit + Tailwind scaffold in `web/`, build embedded via `go:embed`
- [ ] [P] Library grid, movie detail, add-movie flow
- [ ] [P] Activity/queue, history, calendar
- [ ] [P] Settings (profiles, root folders, indexers status, download clients)
- [ ] [P] PWA manifest + offline shell + mobile nav

## Phase 9 — Hardening & release  `[ ]`
- [ ] [P] Backup/restore, health checks, rate limiting
- [ ] [P] OpenAPI spec generation + published docs
- [ ] [P] Observability (metrics/tracing), e2e tests
- [ ] [P] Tagged multi-arch release + published image + install docs

---

### Sub-agent orchestration notes
- Tasks marked **[P]** within a phase can run concurrently in separate worktrees.
- **[S]** tasks gate their phase — land them first.
- Phase 3's test corpus is the highest-value fan-out: spawn many agents to harvest
  and label real release names, then a verifier agent to dedupe and adjudicate.
- Treat each ✅ conformance/daily-driver gate as a hard checkpoint before moving on.
