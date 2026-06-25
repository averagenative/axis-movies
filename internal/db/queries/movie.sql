-- name: ListMovies :many
SELECT * FROM movie ORDER BY sort_title NULLS LAST, title;

-- name: GetMovie :one
SELECT * FROM movie WHERE id = $1;

-- name: GetMovieByTMDB :one
SELECT * FROM movie WHERE tmdb_id = $1;

-- name: CreateMovie :one
INSERT INTO movie (
    tmdb_id, title, year, monitored, title_slug, sort_title, overview,
    status, runtime, imdb_id, path, root_folder_path, images, quality_profile_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;
