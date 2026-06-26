-- name: ListDownloadClients :many
SELECT * FROM download_client ORDER BY priority, name;

-- name: GetDownloadClient :one
SELECT * FROM download_client WHERE id = $1;

-- name: CreateDownloadClient :one
INSERT INTO download_client (
    name, implementation, config_contract, protocol, priority, enable, fields, tags
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateDownloadClient :one
UPDATE download_client SET
    name = $2,
    implementation = $3,
    config_contract = $4,
    protocol = $5,
    priority = $6,
    enable = $7,
    fields = $8,
    tags = $9,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDownloadClient :exec
DELETE FROM download_client WHERE id = $1;
