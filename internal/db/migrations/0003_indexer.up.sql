-- Indexers pushed by Prowlarr (Torznab/Newznab) via the Radarr v3 indexer API.
-- The provider-specific settings (baseUrl, apiKey, categories, ...) are stored
-- verbatim as the Radarr field array in JSONB.

CREATE TABLE IF NOT EXISTS indexer (
    id                        BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                      TEXT        NOT NULL,
    implementation            TEXT        NOT NULL,
    config_contract           TEXT        NOT NULL,
    protocol                  TEXT        NOT NULL DEFAULT 'torrent',
    priority                  INT         NOT NULL DEFAULT 25,
    enable_rss                BOOLEAN     NOT NULL DEFAULT TRUE,
    enable_automatic_search   BOOLEAN     NOT NULL DEFAULT TRUE,
    enable_interactive_search BOOLEAN     NOT NULL DEFAULT TRUE,
    fields                    JSONB       NOT NULL DEFAULT '[]'::jsonb,
    tags                      JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS indexer_protocol_idx ON indexer (protocol);
