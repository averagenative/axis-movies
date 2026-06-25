package tmdb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	searchBody = `{"results":[{"id":603,"title":"The Matrix","release_date":"1999-03-30","overview":"A hacker learns the truth.","poster_path":"/poster.jpg","backdrop_path":"/back.jpg"}]}`
	detailBody = `{"id":603,"imdb_id":"tt0133093","title":"The Matrix","release_date":"1999-03-30","overview":"A hacker learns the truth.","runtime":136,"status":"Released","poster_path":"/poster.jpg","backdrop_path":"/back.jpg"}`
)

func mockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") == "" {
			t.Errorf("missing api_key on %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/movie"):
			_, _ = w.Write([]byte(searchBody))
		case r.URL.Path == "/movie/603":
			_, _ = w.Write([]byte(detailBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestSearch(t *testing.T) {
	srv := mockServer(t)
	defer srv.Close()
	c := New("key", srv.URL, "https://img", srv.Client())

	movies, err := c.Search(context.Background(), "matrix")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(movies) != 1 {
		t.Fatalf("got %d results, want 1", len(movies))
	}
	if movies[0].TMDBID != 603 || movies[0].Title != "The Matrix" || movies[0].Year != 1999 {
		t.Fatalf("unexpected result: %+v", movies[0])
	}
}

func TestGetMovie(t *testing.T) {
	srv := mockServer(t)
	defer srv.Close()
	c := New("key", srv.URL, "https://img", srv.Client())

	m, err := c.GetMovie(context.Background(), 603)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.Runtime != 136 || m.IMDBID != "tt0133093" || m.Status != "Released" {
		t.Fatalf("unexpected details: %+v", m)
	}
}

func TestGetMovieNotFound(t *testing.T) {
	srv := mockServer(t)
	defer srv.Close()
	c := New("key", srv.URL, "https://img", srv.Client())

	if _, err := c.GetMovie(context.Background(), 999999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestDisabledClient(t *testing.T) {
	c := New("", "https://api.themoviedb.org/3", "https://img", nil)
	if c.Enabled() {
		t.Fatal("client should be disabled without an API key")
	}
	if _, err := c.Search(context.Background(), "x"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("got %v, want ErrNotConfigured", err)
	}
}

func TestImageURL(t *testing.T) {
	c := New("key", "https://api.themoviedb.org/3", "https://image.tmdb.org/t/p", nil)
	if got := c.ImageURL("/poster.jpg"); got != "https://image.tmdb.org/t/p/original/poster.jpg" {
		t.Fatalf("image url: %s", got)
	}
	if got := c.ImageURL(""); got != "" {
		t.Fatalf("empty path should give empty url, got %s", got)
	}
}
