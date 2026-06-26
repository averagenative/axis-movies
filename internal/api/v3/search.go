package v3

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/averagenative/axis-movies/internal/parser"
	"github.com/averagenative/axis-movies/internal/quality"
	"github.com/averagenative/axis-movies/internal/store"
	"github.com/averagenative/axis-movies/internal/torznab"
)

// searchReleases performs an interactive search for a movie across all enabled
// indexers (GET /api/v3/release?movieId=). Results are parsed, quality-scored,
// and returned best-first.
func (a *API) searchReleases(w http.ResponseWriter, r *http.Request) {
	movieID, err := strconv.ParseInt(r.URL.Query().Get("movieId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "movieId is required")
		return
	}
	movie, err := a.q.GetMovie(r.Context(), movieID)
	if isNotFound(err) {
		writeError(w, http.StatusNotFound, "movie not found")
		return
	}
	if err != nil {
		a.serverError(w, err)
		return
	}
	indexers, err := a.q.ListIndexers(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}

	term := movie.Title
	if movie.Year.Valid && movie.Year.Int32 > 0 {
		term += " " + strconv.Itoa(int(movie.Year.Int32))
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	var (
		mu  sync.Mutex
		all []map[string]any
		wg  sync.WaitGroup
	)
	for _, ix := range indexers {
		if !ix.EnableInteractiveSearch {
			continue
		}
		wg.Add(1)
		go func(ix store.Indexer) {
			defer wg.Done()
			items := a.searchIndexer(ctx, ix, term)
			local := make([]map[string]any, 0, len(items))
			for _, it := range items {
				local = append(local, releaseJSON(it, ix, movieID))
			}
			mu.Lock()
			all = append(all, local...)
			mu.Unlock()
		}(ix)
	}
	wg.Wait()

	sort.SliceStable(all, func(i, j int) bool {
		wi, wj := all[i]["qualityWeight"].(int), all[j]["qualityWeight"].(int)
		if wi != wj {
			return wi > wj
		}
		return all[i]["seeders"].(int) > all[j]["seeders"].(int)
	})
	writeJSON(w, http.StatusOK, all)
}

func (a *API) searchIndexer(ctx context.Context, ix store.Indexer, term string) []torznab.Item {
	fields := parseIndexerFields(ix.Fields)
	url := torznab.SearchURL(
		fieldString(fields, "baseUrl"),
		fieldString(fields, "apiPath"),
		fieldString(fields, "apiKey"),
		term,
		fieldInts(fields, "categories"),
	)
	items, err := a.tz.Search(ctx, url)
	if err != nil {
		a.log.Warn("indexer search failed", "indexer", ix.Name, "err", err)
		return nil
	}
	return items
}

func releaseJSON(it torznab.Item, ix store.Indexer, movieID int64) map[string]any {
	rel := parser.Parse(it.Title)
	qName := quality.Name(rel.Source, rel.Resolution)
	qWeight := quality.Weight(rel.Source, rel.Resolution)

	guid := it.MagnetURL
	if guid == "" {
		guid = it.DownloadURL
	}
	if guid == "" {
		guid = it.Title
	}

	return map[string]any{
		"guid":          guid,
		"title":         it.Title,
		"size":          it.Size,
		"indexerId":     ix.ID,
		"indexer":       ix.Name,
		"seeders":       it.Seeders,
		"leechers":      it.Leechers,
		"protocol":      ix.Protocol,
		"downloadUrl":   it.DownloadURL,
		"magnetUrl":     it.MagnetURL,
		"infoUrl":       it.InfoURL,
		"movieId":       movieID,
		"qualityWeight": qWeight,
		"rejected":      false,
		"quality": map[string]any{
			"quality": map[string]any{
				"name":       qName,
				"resolution": resolutionInt(rel.Resolution),
				"source":     strings.ToLower(rel.Source),
			},
			"revision": map[string]any{"version": 1},
		},
		// Axis-native parse detail (ignored by Radarr clients).
		"axisParsed": map[string]any{
			"title": rel.Title, "year": rel.Year, "codec": rel.Codec,
			"group": rel.Group, "proper": rel.Proper, "repack": rel.Repack,
		},
	}
}

type indexerField struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

func parseIndexerFields(b []byte) []indexerField {
	var f []indexerField
	_ = json.Unmarshal(b, &f)
	return f
}

func fieldString(fs []indexerField, name string) string {
	for _, f := range fs {
		if strings.EqualFold(f.Name, name) {
			var s string
			if json.Unmarshal(f.Value, &s) == nil {
				return s
			}
		}
	}
	return ""
}

func fieldInts(fs []indexerField, name string) []int {
	for _, f := range fs {
		if strings.EqualFold(f.Name, name) {
			var a []int
			if json.Unmarshal(f.Value, &a) == nil {
				return a
			}
		}
	}
	return nil
}

func resolutionInt(res string) int {
	switch res {
	case "2160p":
		return 2160
	case "1080p":
		return 1080
	case "720p":
		return 720
	case "576p":
		return 576
	case "480p":
		return 480
	default:
		return 0
	}
}
