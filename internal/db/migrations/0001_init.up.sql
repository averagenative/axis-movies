-- Initial Axis schema baseline.
-- This is intentionally minimal for Phase 0; the movie domain model is fleshed
-- out in Phase 1 (see TASKS.md).

CREATE TABLE IF NOT EXISTS root_folder (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    path        TEXT        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS movie (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tmdb_id     BIGINT      NOT NULL UNIQUE,
    title       TEXT        NOT NULL,
    year        INT,
    monitored   BOOLEAN     NOT NULL DEFAULT TRUE,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS movie_title_idx ON movie (title);
