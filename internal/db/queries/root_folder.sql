-- name: ListRootFolders :many
SELECT * FROM root_folder ORDER BY path;

-- name: GetRootFolder :one
SELECT * FROM root_folder WHERE id = $1;

-- name: CreateRootFolder :one
INSERT INTO root_folder (path) VALUES ($1)
RETURNING *;

-- name: DeleteRootFolder :exec
DELETE FROM root_folder WHERE id = $1;
