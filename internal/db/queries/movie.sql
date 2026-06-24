-- name: ListMovies :many
SELECT * FROM movie ORDER BY sort_title NULLS LAST, title;

-- name: GetMovie :one
SELECT * FROM movie WHERE id = $1;

-- name: GetMovieByTMDB :one
SELECT * FROM movie WHERE tmdb_id = $1;
