-- name: ListIndexers :many
SELECT * FROM indexer ORDER BY name;

-- name: GetIndexer :one
SELECT * FROM indexer WHERE id = $1;

-- name: CreateIndexer :one
INSERT INTO indexer (
    name, implementation, config_contract, protocol, priority,
    enable_rss, enable_automatic_search, enable_interactive_search, fields, tags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: UpdateIndexer :one
UPDATE indexer SET
    name = $2,
    implementation = $3,
    config_contract = $4,
    protocol = $5,
    priority = $6,
    enable_rss = $7,
    enable_automatic_search = $8,
    enable_interactive_search = $9,
    fields = $10,
    tags = $11,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteIndexer :exec
DELETE FROM indexer WHERE id = $1;
