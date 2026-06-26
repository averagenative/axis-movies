package v3

import (
	"net/http"

	"github.com/averagenative/axis-movies/internal/download"
	"github.com/averagenative/axis-movies/internal/store"
)

type grabRequest struct {
	GUID        string `json:"guid"`
	IndexerID   int64  `json:"indexerId"`
	Title       string `json:"title"`
	DownloadURL string `json:"downloadUrl"`
	MagnetURL   string `json:"magnetUrl"`
	Protocol    string `json:"protocol"`
	MovieID     int64  `json:"movieId"`
}

// grabRelease sends a chosen release to a matching download client
// (POST /api/v3/release).
func (a *API) grabRelease(w http.ResponseWriter, r *http.Request) {
	var b grabRequest
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if b.DownloadURL == "" && b.MagnetURL == "" {
		writeError(w, http.StatusBadRequest, "release has no download URL or magnet")
		return
	}
	proto := b.Protocol
	if proto == "" {
		proto = "torrent" // usenet must be explicit; default to torrent
	}

	clients, err := a.q.ListDownloadClients(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	var chosen *store.DownloadClient
	for i := range clients {
		if clients[i].Enable && clients[i].Protocol == proto {
			chosen = &clients[i]
			break
		}
	}
	if chosen == nil {
		writeError(w, http.StatusConflict, "no enabled download client for protocol "+proto)
		return
	}

	client, err := download.New(clientConfig(chosen.Implementation, parseIndexerFields(chosen.Fields)), nil)
	if err != nil {
		a.serverError(w, err)
		return
	}
	rel := download.Release{Title: b.Title, DownloadURL: b.DownloadURL, MagnetURL: b.MagnetURL, Protocol: proto}
	if err := client.Add(r.Context(), rel); err != nil {
		a.log.Error("grab failed", "client", chosen.Name, "title", b.Title, "err", err)
		writeError(w, http.StatusBadGateway, "download client error: "+err.Error())
		return
	}
	a.log.Info("grabbed release", "title", b.Title, "client", chosen.Name, "protocol", proto)
	writeJSON(w, http.StatusCreated, map[string]any{
		"guid":           b.GUID,
		"title":          b.Title,
		"movieId":        b.MovieID,
		"protocol":       proto,
		"downloadClient": chosen.Name,
		"status":         "grabbed",
	})
}
