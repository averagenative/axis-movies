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
- [x] CI (gofmt, vet, race tests, build, golangci-lint, image build)
- [ ] [P] Add `sqlc` config + generate typed query layer scaffolding
- [ ] [P] Multi-arch release workflow (goreleaser or buildx amd64+arm64) + GHCR push
- [ ] [P] `/system/status` integration test against a live Postgres (testcontainers)

## Phase 1 — v3 API: read surface (real data)  `[ ]`
- [ ] [S] Domain models + migrations: movie, root_folder, quality_profile, tag, history, blocklist
- [ ] [P] `GET/POST/PUT/DELETE /api/v3/rootfolder` (DB-backed)
- [ ] [P] `GET/POST/DELETE /api/v3/tag`
- [ ] [P] `GET /api/v3/movie` + `GET /api/v3/movie/{id}` (DB-backed)
- [ ] [P] `GET /api/v3/qualityprofile` (DB-backed defaults seeded)
- [ ] [S] Pagination/sort envelope matching Radarr (`/api/v3/movie/lookup` later)
- [ ] ✅ **Conformance gate:** Prowlarr successfully adds this as a "Radarr" app; Overseerr & nzb360 connect read-only

## Phase 2 — Metadata  `[ ]`
- [ ] [S] TMDb client (own API key) + Postgres response cache
- [ ] [P] `GET /api/v3/movie/lookup?term=` (TMDb-backed search)
- [ ] [P] `POST /api/v3/movie` (add by tmdbId), image proxy/cache
- [ ] [P] Refresh-movie metadata job

## Phase 3 — Release parser (crown jewels)  `[ ]`
- [ ] [S] Clean-room parser: title, year, resolution, source, codec, audio, group, edition, proper/repack, language
- [ ] [P] **Test corpus** (great sub-agent fan-out: harvest real release names, label expected fields)
- [ ] [P] Property/regression tests vs corpus; quarantine ambiguous cases
- [ ] [P] Scene/anime/repack edge-case coverage

## Phase 4 — Decision engine + indexer ingestion  `[ ]`
- [ ] [S] Wire **River** (Postgres-backed job queue); migrate scheduler onto it
- [ ] [S] Torznab/Newznab feed consumer (config pushed by Prowlarr via v3 indexer endpoints)
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
- [ ] [P] Tagged multi-arch release + GHCR image + install docs

---

### Sub-agent orchestration notes
- Tasks marked **[P]** within a phase can run concurrently in separate worktrees.
- **[S]** tasks gate their phase — land them first.
- Phase 3's test corpus is the highest-value fan-out: spawn many agents to harvest
  and label real release names, then a verifier agent to dedupe and adjudicate.
- Treat each ✅ conformance/daily-driver gate as a hard checkpoint before moving on.
