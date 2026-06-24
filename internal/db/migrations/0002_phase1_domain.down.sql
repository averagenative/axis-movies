DROP INDEX IF EXISTS movie_title_slug_idx;

ALTER TABLE movie
    DROP COLUMN IF EXISTS title_slug,
    DROP COLUMN IF EXISTS sort_title,
    DROP COLUMN IF EXISTS overview,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS runtime,
    DROP COLUMN IF EXISTS has_file,
    DROP COLUMN IF EXISTS imdb_id,
    DROP COLUMN IF EXISTS path,
    DROP COLUMN IF EXISTS root_folder_path,
    DROP COLUMN IF EXISTS images,
    DROP COLUMN IF EXISTS quality_profile_id,
    DROP COLUMN IF EXISTS updated_at;

DROP TABLE IF EXISTS tag;
DROP TABLE IF EXISTS quality_profile;
