package v3

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/averagenative/axis-movies/internal/store"
	"github.com/averagenative/axis-movies/internal/tmdb"
)

// lookupMovie searches TMDb for movies matching ?term=.
func (a *API) lookupMovie(w http.ResponseWriter, r *http.Request) {
	term := strings.TrimSpace(r.URL.Query().Get("term"))
	if term == "" {
		writeError(w, http.StatusBadRequest, "term is required")
		return
	}
	movies, err := a.tmdb.Search(r.Context(), term)
	if err != nil {
		a.tmdbError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(movies))
	for _, m := range movies {
		out = append(out, a.lookupJSON(m))
	}
	writeJSON(w, http.StatusOK, out)
}

// addMovie adds a movie to the library by TMDb id, fetching its metadata.
func (a *API) addMovie(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TmdbID           int64  `json:"tmdbId"`
		QualityProfileID int64  `json:"qualityProfileId"`
		RootFolderPath   string `json:"rootFolderPath"`
		Monitored        *bool  `json:"monitored"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.TmdbID == 0 {
		writeError(w, http.StatusBadRequest, "tmdbId is required")
		return
	}

	// Reject duplicates up front for a clean 409.
	if _, err := a.q.GetMovieByTMDB(r.Context(), body.TmdbID); err == nil {
		writeError(w, http.StatusConflict, "movie already exists")
		return
	} else if !isNotFound(err) {
		a.serverError(w, err)
		return
	}

	m, err := a.tmdb.GetMovie(r.Context(), body.TmdbID)
	if errors.Is(err, tmdb.ErrNotFound) {
		writeError(w, http.StatusNotFound, "movie not found on TMDb")
		return
	}
	if err != nil {
		a.tmdbError(w, err)
		return
	}

	monitored := true
	if body.Monitored != nil {
		monitored = *body.Monitored
	}
	qpid := body.QualityProfileID
	if qpid == 0 {
		qpid = 1 // seeded default "Any" profile
	}

	images, _ := json.Marshal(a.tmdbImages(m))
	var path string
	if body.RootFolderPath != "" {
		path = strings.TrimRight(body.RootFolderPath, "/") + "/" + folderName(m)
	}

	created, err := a.q.CreateMovie(r.Context(), store.CreateMovieParams{
		TmdbID:           m.TMDBID,
		Title:            m.Title,
		Year:             pgInt4(int32(m.Year)),
		Monitored:        monitored,
		TitleSlug:        pgText(movieSlug(m)),
		SortTitle:        pgText(strings.ToLower(m.Title)),
		Overview:         pgText(m.Overview),
		Status:           pgText(m.Status),
		Runtime:          int32(m.Runtime),
		ImdbID:           pgText(m.IMDBID),
		Path:             pgText(path),
		RootFolderPath:   pgText(body.RootFolderPath),
		Images:           images,
		QualityProfileID: pgInt8(qpid),
	})
	switch {
	case isUniqueViolation(err):
		writeError(w, http.StatusConflict, "movie already exists")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusCreated, movieJSON(created))
	}
}

// tmdbError maps TMDb client errors to HTTP responses.
func (a *API) tmdbError(w http.ResponseWriter, err error) {
	if errors.Is(err, tmdb.ErrNotConfigured) {
		writeError(w, http.StatusServiceUnavailable, "metadata unavailable: set AXIS_TMDB_API_KEY")
		return
	}
	a.log.Error("tmdb error", "err", err)
	writeError(w, http.StatusBadGateway, "metadata provider error")
}

// lookupJSON maps a TMDb movie to the Radarr v3 lookup result shape.
func (a *API) lookupJSON(m tmdb.Movie) map[string]any {
	return map[string]any{
		"title":            m.Title,
		"tmdbId":           m.TMDBID,
		"year":             m.Year,
		"overview":         m.Overview,
		"runtime":          m.Runtime,
		"status":           m.Status,
		"imdbId":           m.IMDBID,
		"titleSlug":        movieSlug(m),
		"images":           a.tmdbImages(m),
		"monitored":        false,
		"hasFile":          false,
		"qualityProfileId": 0,
	}
}

func (a *API) tmdbImages(m tmdb.Movie) []map[string]any {
	imgs := []map[string]any{}
	if u := a.tmdb.ImageURL(m.PosterPath); u != "" {
		imgs = append(imgs, map[string]any{"coverType": "poster", "remoteUrl": u})
	}
	if u := a.tmdb.ImageURL(m.BackdropPath); u != "" {
		imgs = append(imgs, map[string]any{"coverType": "fanart", "remoteUrl": u})
	}
	return imgs
}

func movieSlug(m tmdb.Movie) string {
	slug := slugify(m.Title)
	if slug == "" {
		slug = "movie"
	}
	return slug + "-" + strconv.FormatInt(m.TMDBID, 10)
}

func folderName(m tmdb.Movie) string {
	name := strings.ReplaceAll(m.Title, "/", "-")
	if m.Year > 0 {
		name += " (" + strconv.Itoa(m.Year) + ")"
	}
	return name
}
