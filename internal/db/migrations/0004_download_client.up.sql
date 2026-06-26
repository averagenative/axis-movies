-- Download clients (qBittorrent, SABnzbd, ...) configured directly in Axis.
-- Provider settings (host, port, credentials, category) are stored as the Radarr
-- field array in JSONB.

CREATE TABLE IF NOT EXISTS download_client (
    id              BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name            TEXT        NOT NULL,
    implementation  TEXT        NOT NULL,
    config_contract TEXT        NOT NULL,
    protocol        TEXT        NOT NULL DEFAULT 'torrent',
    priority        INT         NOT NULL DEFAULT 1,
    enable          BOOLEAN     NOT NULL DEFAULT TRUE,
    fields          JSONB       NOT NULL DEFAULT '[]'::jsonb,
    tags            JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS download_client_protocol_idx ON download_client (protocol);
