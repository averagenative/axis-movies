# Tasks — Bootstrap

See `TASKS.md` at the repo root for the full phased roadmap. This change covers
Phase 0 and the entry into Phase 1.

## Phase 0
- [x] Go module, GPLv3, README, repo
- [x] Config (YAML + env), logging
- [x] Postgres pool + embedded migrations
- [x] chi router, API-key auth, `/ping`
- [x] v3 read stubs
- [x] Docker + compose + local `make check` gate (no GitHub Actions)
- [x] sqlc scaffolding (`internal/store`)
- [ ] multi-arch release build (local/manual) + image publish
- [ ] live-Postgres integration test

## Phase 1 entry
- [x] DB-backed movie/rootfolder/tag/qualityprofile models (verified vs Postgres)
- [ ] Conformance gate: Prowlarr adds Axis as a "Radarr" app (needs live Prowlarr)
