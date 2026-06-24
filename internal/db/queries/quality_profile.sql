-- name: ListQualityProfiles :many
SELECT * FROM quality_profile ORDER BY id;

-- name: GetQualityProfile :one
SELECT * FROM quality_profile WHERE id = $1;
