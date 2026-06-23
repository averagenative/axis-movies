# Design — Bootstrap

## Context
First commit must produce a compiling, runnable foundation that ecosystem tools
can eventually talk to, without prematurely stubbing large frameworks.

## Decisions

### Radarr v3 as the single public contract
Implement Radarr's v3 API rather than a bespoke one. Trade-off: we inherit some
crusty shapes, but gain the entire ecosystem (Prowlarr push, Overseerr, mobile).
Native extensions, if needed, are additive under a separate versioned prefix —
never a second parallel contract.

### Compatibility identity
`GET /system/status` advertises `appName: "Radarr"` and a Radarr-compatible
version (configurable via `compat_app_name`), with true identity in `axis*`
fields. Required because ecosystem tools string-match the app type.

### Postgres-first, no SQLite
Single SQL dialect; schema via embedded golang-migrate. Removes the main source
of the corruption problems that motivate the project.

### Defer River and sqlc
Phase 0 wires chi + pgx + migrations only. River (Postgres-backed jobs) and sqlc
(typed queries) are introduced in the phases that first need them (4 and 1),
keeping the foundation honest and compiling rather than carrying unused scaffolding.

## Risks
- **Release parser** (Phase 3) is the hardest part; mitigated by a large labeled
  test corpus built via sub-agent fan-out.
- **API drift** vs real Radarr; mitigated by conformance testing against live
  Prowlarr/Overseerr at the Phase 1 gate.

## Open questions
- TMDb API key provisioning + rate-limit strategy (Phase 2).
- Whether any native endpoints are needed for the PWA beyond v3 (revisit Phase 8).
