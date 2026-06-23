# Bootstrap Axis Movies

## Why
There is no Go, Radarr v3 API-compatible, Postgres-first movie manager. Existing
Go efforts use proprietary APIs (so Prowlarr/Overseerr/mobile clients don't work),
and Radarr's SQLite default is corruption-prone on container/network filesystems.
We want a reliable, container-native, mobile-friendly drop-in.

## What Changes
- Establish the Go service foundation: config, logging, HTTP server, Postgres
  pool, embedded migrations, API-key auth, CI, and container packaging.
- Stand up the **Radarr v3 read API** surface (initially stubbed, then DB-backed)
  as the single public contract.
- Commit to Postgres-first storage, Prowlarr-delegated indexers, and a SvelteKit
  PWA — captured as specs and the phased plan in `TASKS.md`.

## Impact
- New repository `averagenative/axis-movies` (Go, GPLv3).
- Affected capabilities: `api-compat` (new), `platform` (new).
- Establishes the conformance gate: Prowlarr can add Axis as a "Radarr" app.
