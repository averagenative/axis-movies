-- Imported movie files and the activity history (grabs, imports).

CREATE TABLE IF NOT EXISTS movie_file (
    id            BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    movie_id      BIGINT      NOT NULL UNIQUE REFERENCES movie (id) ON DELETE CASCADE,
    relative_path TEXT        NOT NULL,
    path          TEXT        NOT NULL,
    size          BIGINT      NOT NULL DEFAULT 0,
    quality       TEXT        NOT NULL DEFAULT '',
    date_added    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS history (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    movie_id     BIGINT      REFERENCES movie (id) ON DELETE CASCADE,
    event_type   TEXT        NOT NULL,
    source_title TEXT        NOT NULL DEFAULT '',
    quality      TEXT        NOT NULL DEFAULT '',
    data         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    date         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS history_movie_idx ON history (movie_id);
CREATE INDEX IF NOT EXISTS history_date_idx ON history (date DESC);
