-- Phase 1 domain model: quality profiles, tags, and an expanded movie resource
-- sufficient to render a Radarr v3 read surface and pass the Prowlarr
-- "add as Radarr application" conformance gate.
--
-- history and blocklist tables are intentionally deferred to the phases that
-- first expose them (grab/import); creating them now would be unused.

CREATE TABLE IF NOT EXISTS quality_profile (
    id                BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name              TEXT        NOT NULL UNIQUE,
    upgrade_allowed   BOOLEAN     NOT NULL DEFAULT TRUE,
    cutoff_quality_id INT,
    items             JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tag (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    label      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Expand the movie resource with the fields Radarr clients expect to render.
ALTER TABLE movie
    ADD COLUMN IF NOT EXISTS title_slug         TEXT,
    ADD COLUMN IF NOT EXISTS sort_title         TEXT,
    ADD COLUMN IF NOT EXISTS overview           TEXT,
    ADD COLUMN IF NOT EXISTS status             TEXT,
    ADD COLUMN IF NOT EXISTS runtime            INT     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS has_file           BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS imdb_id            TEXT,
    ADD COLUMN IF NOT EXISTS path               TEXT,
    ADD COLUMN IF NOT EXISTS root_folder_path   TEXT,
    ADD COLUMN IF NOT EXISTS images             JSONB   NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS quality_profile_id BIGINT  REFERENCES quality_profile (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS updated_at         TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE UNIQUE INDEX IF NOT EXISTS movie_title_slug_idx ON movie (title_slug);

-- Seed a usable default profile so /api/v3/qualityprofile returns real data and
-- newly added movies have a profile to point at.
INSERT INTO quality_profile (name, upgrade_allowed, items)
VALUES ('Any', TRUE, '[]'::jsonb)
ON CONFLICT (name) DO NOTHING;
