-- name: ListTags :many
SELECT * FROM tag ORDER BY label;

-- name: GetTag :one
SELECT * FROM tag WHERE id = $1;

-- name: CreateTag :one
INSERT INTO tag (label) VALUES ($1)
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tag WHERE id = $1;
