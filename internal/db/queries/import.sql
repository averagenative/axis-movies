-- name: UpsertMovieFile :one
INSERT INTO movie_file (movie_id, relative_path, path, size, quality)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (movie_id) DO UPDATE SET
    relative_path = EXCLUDED.relative_path,
    path = EXCLUDED.path,
    size = EXCLUDED.size,
    quality = EXCLUDED.quality,
    date_added = now()
RETURNING *;

-- name: GetMovieFile :one
SELECT * FROM movie_file WHERE movie_id = $1;

-- name: SetMovieImported :exec
UPDATE movie SET has_file = TRUE, path = $2, updated_at = now() WHERE id = $1;

-- name: CreateHistory :one
INSERT INTO history (movie_id, event_type, source_title, quality, data)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListHistory :many
SELECT * FROM history ORDER BY date DESC LIMIT $1;
