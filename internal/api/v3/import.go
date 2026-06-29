package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/averagenative/axis-movies/internal/importer"
	"github.com/averagenative/axis-movies/internal/parser"
	"github.com/averagenative/axis-movies/internal/quality"
	"github.com/averagenative/axis-movies/internal/store"
)

type commandRequest struct {
	Name    string `json:"name"`
	MovieID int64  `json:"movieId"`
	Path    string `json:"path"`
}

// runCommand handles POST /api/v3/command. Import-related commands trigger an
// import of the given path into the movie's library folder; other Radarr
// commands are accepted as no-ops so clients don't error.
func (a *API) runCommand(w http.ResponseWriter, r *http.Request) {
	var b commandRequest
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	switch strings.ToLower(b.Name) {
	case "downloadedmoviesscan", "manualimport", "processmonitoreddownloads":
		if b.MovieID == 0 || b.Path == "" {
			writeError(w, http.StatusBadRequest, "movieId and path are required for import")
			return
		}
		movie, err := a.q.GetMovie(r.Context(), b.MovieID)
		if isNotFound(err) {
			writeError(w, http.StatusNotFound, "movie not found")
			return
		}
		if err != nil {
			a.serverError(w, err)
			return
		}
		dest, err := a.importMovie(r.Context(), movie, b.Path)
		if err != nil {
			a.log.Error("import failed", "movie", movie.Title, "err", err)
			writeError(w, http.StatusBadRequest, "import failed: "+err.Error())
			return
		}
		a.log.Info("imported movie", "movie", movie.Title, "dest", dest)
		writeJSON(w, http.StatusCreated, map[string]any{
			"name": b.Name, "status": "completed", "result": "successful", "importedPath": dest,
		})
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"name": b.Name, "status": "completed"})
	}
}

// importMovie scans sourcePath for the feature file, parses it, and hardlinks it
// into the movie's library folder, recording the movie file and history.
func (a *API) importMovie(ctx context.Context, movie store.Movie, sourcePath string) (string, error) {
	files, err := importer.ScanVideoFiles(sourcePath)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no video files found in %q", sourcePath)
	}
	main := files[0]

	parsed := parser.Parse(filepath.Base(main.Path))
	qName := quality.Name(parsed.Source, parsed.Resolution)

	year := 0
	if movie.Year.Valid {
		year = int(movie.Year.Int32)
	}
	root := a.rootFolderFor(ctx, movie)
	if root == "" {
		return "", fmt.Errorf("no root folder configured")
	}

	ext := filepath.Ext(main.Path)
	dest := importer.DestPath(root, movie.Title, year, qName, ext)
	if err := importer.Import(main.Path, dest); err != nil {
		return "", err
	}

	folder := importer.FolderName(movie.Title, year)
	relPath := filepath.Join(folder, filepath.Base(dest))
	if _, err := a.q.UpsertMovieFile(ctx, store.UpsertMovieFileParams{
		MovieID: movie.ID, RelativePath: relPath, Path: dest, Size: main.Size, Quality: qName,
	}); err != nil {
		return "", err
	}
	if err := a.q.SetMovieImported(ctx, store.SetMovieImportedParams{
		ID: movie.ID, Path: pgText(filepath.Join(root, folder)),
	}); err != nil {
		return "", err
	}
	data, _ := json.Marshal(map[string]any{"droppedPath": main.Path, "importedPath": dest})
	_, _ = a.q.CreateHistory(ctx, store.CreateHistoryParams{
		MovieID: pgInt8(movie.ID), EventType: "downloadFolderImported",
		SourceTitle: filepath.Base(main.Path), Quality: qName, Data: data,
	})
	return dest, nil
}

func (a *API) rootFolderFor(ctx context.Context, movie store.Movie) string {
	if movie.RootFolderPath.Valid && movie.RootFolderPath.String != "" {
		return movie.RootFolderPath.String
	}
	if folders, err := a.q.ListRootFolders(ctx); err == nil && len(folders) > 0 {
		return folders[0].Path
	}
	return ""
}

func (a *API) listHistory(w http.ResponseWriter, r *http.Request) {
	rows, err := a.q.ListHistory(r.Context(), 100)
	if err != nil {
		a.serverError(w, err)
		return
	}
	records := make([]map[string]any, 0, len(rows))
	for _, h := range rows {
		records = append(records, map[string]any{
			"id":          h.ID,
			"movieId":     int8Val(h.MovieID),
			"eventType":   h.EventType,
			"sourceTitle": h.SourceTitle,
			"date":        tsVal(h.Date),
			"data":        rawJSON(h.Data),
			"quality":     map[string]any{"quality": map[string]any{"name": h.Quality}},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"page": 1, "pageSize": 100, "totalRecords": len(records), "records": records,
	})
}

func (a *API) listMovieFiles(w http.ResponseWriter, r *http.Request) {
	out := make([]map[string]any, 0, 1)
	if mid, err := strconv.ParseInt(r.URL.Query().Get("movieId"), 10, 64); err == nil {
		mf, err := a.q.GetMovieFile(r.Context(), mid)
		switch {
		case isNotFound(err):
			// no file yet; empty array
		case err != nil:
			a.serverError(w, err)
			return
		default:
			out = append(out, movieFileJSON(mf))
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func movieFileJSON(mf store.MovieFile) map[string]any {
	return map[string]any{
		"id":           mf.ID,
		"movieId":      mf.MovieID,
		"relativePath": mf.RelativePath,
		"path":         mf.Path,
		"size":         mf.Size,
		"dateAdded":    tsVal(mf.DateAdded),
		"quality":      map[string]any{"quality": map[string]any{"name": mf.Quality}},
	}
}
