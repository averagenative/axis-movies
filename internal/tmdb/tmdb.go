// Package tmdb is a small client for the themoviedb.org v3 API, used for movie
// metadata lookup. Axis brings its own TMDb API key (it cannot use Radarr's
// api.radarr.video proxy). Response caching is a later increment; this client
// hits TMDb directly.
package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrNotConfigured is returned when no TMDb API key is set.
var ErrNotConfigured = errors.New("tmdb: no API key configured")

// ErrNotFound is returned when TMDb has no movie for the given id.
var ErrNotFound = errors.New("tmdb: movie not found")

// Movie is the normalized subset of TMDb data Axis stores/serves.
type Movie struct {
	TMDBID       int64
	IMDBID       string
	Title        string
	Year         int
	Overview     string
	Runtime      int
	Status       string
	ReleaseDate  string
	PosterPath   string
	BackdropPath string
}

// Client talks to the TMDb v3 API.
type Client struct {
	apiKey       string
	baseURL      string
	imageBaseURL string
	http         *http.Client
}

// New builds a TMDb client. A nil httpClient gets a sane default.
func New(apiKey, baseURL, imageBaseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		apiKey:       apiKey,
		baseURL:      strings.TrimRight(baseURL, "/"),
		imageBaseURL: strings.TrimRight(imageBaseURL, "/"),
		http:         httpClient,
	}
}

// Enabled reports whether metadata lookups are possible (an API key is set).
func (c *Client) Enabled() bool { return c.apiKey != "" }

// ImageURL builds an absolute TMDb image URL for a poster/backdrop path.
func (c *Client) ImageURL(path string) string {
	if path == "" {
		return ""
	}
	return c.imageBaseURL + "/original" + path
}

type searchResponse struct {
	Results []moviePayload `json:"results"`
}

type moviePayload struct {
	ID           int64  `json:"id"`
	IMDBID       string `json:"imdb_id"`
	Title        string `json:"title"`
	ReleaseDate  string `json:"release_date"`
	Overview     string `json:"overview"`
	Runtime      int    `json:"runtime"`
	Status       string `json:"status"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
}

func (p moviePayload) toMovie() Movie {
	return Movie{
		TMDBID:       p.ID,
		IMDBID:       p.IMDBID,
		Title:        p.Title,
		Year:         yearFromDate(p.ReleaseDate),
		Overview:     p.Overview,
		Runtime:      p.Runtime,
		Status:       p.Status,
		ReleaseDate:  p.ReleaseDate,
		PosterPath:   p.PosterPath,
		BackdropPath: p.BackdropPath,
	}
}

// Search returns movies matching the free-text term.
func (c *Client) Search(ctx context.Context, term string) ([]Movie, error) {
	if !c.Enabled() {
		return nil, ErrNotConfigured
	}
	q := url.Values{}
	q.Set("query", term)
	q.Set("include_adult", "false")

	var resp searchResponse
	if err := c.get(ctx, "/search/movie", q, &resp); err != nil {
		return nil, err
	}
	movies := make([]Movie, 0, len(resp.Results))
	for _, r := range resp.Results {
		movies = append(movies, r.toMovie())
	}
	return movies, nil
}

// GetMovie fetches full details for a single TMDb movie id.
func (c *Client) GetMovie(ctx context.Context, tmdbID int64) (Movie, error) {
	if !c.Enabled() {
		return Movie{}, ErrNotConfigured
	}
	var p moviePayload
	if err := c.get(ctx, "/movie/"+strconv.FormatInt(tmdbID, 10), nil, &p); err != nil {
		return Movie{}, err
	}
	return p.toMovie(), nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, dst any) error {
	if q == nil {
		q = url.Values{}
	}
	q.Set("api_key", c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("tmdb request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("tmdb: unauthorized (check API key)")
	case resp.StatusCode >= 300:
		return fmt.Errorf("tmdb: unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("tmdb decode: %w", err)
	}
	return nil
}

func yearFromDate(date string) int {
	if len(date) < 4 {
		return 0
	}
	y, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return y
}
