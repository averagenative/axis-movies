package v3

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/averagenative/axis-movies/internal/download"
	"github.com/averagenative/axis-movies/internal/store"
)

//go:embed assets/downloadclient_schema.json
var downloadClientSchemaJSON []byte

func (a *API) downloadClientSchema(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(downloadClientSchemaJSON)
}

// downloadClientTest tests connectivity to the posted download client.
func (a *API) downloadClientTest(w http.ResponseWriter, r *http.Request) {
	var b downloadClientRequest
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	client, err := download.New(clientConfig(b.Implementation, parseIndexerFields(rawOrEmpty(b.Fields))), nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := client.TestConnection(r.Context()); err != nil {
		a.log.Warn("download client test failed", "err", err)
		writeJSON(w, http.StatusBadRequest, []map[string]any{
			{"isWarning": false, "propertyName": "host", "errorMessage": err.Error(), "severity": "error"},
		})
		return
	}
	w.WriteHeader(http.StatusOK)
}

type downloadClientRequest struct {
	Name           string          `json:"name"`
	Implementation string          `json:"implementation"`
	ConfigContract string          `json:"configContract"`
	Protocol       string          `json:"protocol"`
	Priority       int32           `json:"priority"`
	Enable         *bool           `json:"enable"`
	Fields         json.RawMessage `json:"fields"`
	Tags           json.RawMessage `json:"tags"`
}

func (a *API) listDownloadClients(w http.ResponseWriter, r *http.Request) {
	rows, err := a.q.ListDownloadClients(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, dc := range rows {
		out = append(out, downloadClientJSON(dc))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getDownloadClient(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	dc, err := a.q.GetDownloadClient(r.Context(), id)
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "download client not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, downloadClientJSON(dc))
	}
}

func (a *API) createDownloadClient(w http.ResponseWriter, r *http.Request) {
	var b downloadClientRequest
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if b.Name == "" || b.Implementation == "" {
		writeError(w, http.StatusBadRequest, "name and implementation are required")
		return
	}
	dc, err := a.q.CreateDownloadClient(r.Context(), store.CreateDownloadClientParams{
		Name:           b.Name,
		Implementation: b.Implementation,
		ConfigContract: b.ConfigContract,
		Protocol:       protocolFor(b.Implementation, b.Protocol),
		Priority:       priorityOr(b.Priority),
		Enable:         boolOr(b.Enable, true),
		Fields:         rawOrEmpty(b.Fields),
		Tags:           rawOrEmpty(b.Tags),
	})
	if err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, downloadClientJSON(dc))
}

func (a *API) updateDownloadClient(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var b downloadClientRequest
	if err := decodeJSON(r, &b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	dc, err := a.q.UpdateDownloadClient(r.Context(), store.UpdateDownloadClientParams{
		ID:             id,
		Name:           b.Name,
		Implementation: b.Implementation,
		ConfigContract: b.ConfigContract,
		Protocol:       protocolFor(b.Implementation, b.Protocol),
		Priority:       priorityOr(b.Priority),
		Enable:         boolOr(b.Enable, true),
		Fields:         rawOrEmpty(b.Fields),
		Tags:           rawOrEmpty(b.Tags),
	})
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "download client not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, downloadClientJSON(dc))
	}
}

func (a *API) deleteDownloadClient(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := a.q.DeleteDownloadClient(r.Context(), id); err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

func downloadClientJSON(dc store.DownloadClient) map[string]any {
	return map[string]any{
		"id":                 dc.ID,
		"name":               dc.Name,
		"implementation":     dc.Implementation,
		"implementationName": dc.Implementation,
		"configContract":     dc.ConfigContract,
		"protocol":           dc.Protocol,
		"priority":           dc.Priority,
		"enable":             dc.Enable,
		"fields":             rawJSON(dc.Fields),
		"tags":               rawJSON(dc.Tags),
	}
}

// clientConfig maps a Radarr field array to a download.Config.
func clientConfig(impl string, fields []indexerField) download.Config {
	return download.Config{
		Implementation: impl,
		Host:           fieldString(fields, "host"),
		Port:           fieldInt(fields, "port"),
		UseSsl:         fieldBool(fields, "useSsl"),
		URLBase:        fieldString(fields, "urlBase"),
		Username:       fieldString(fields, "username"),
		Password:       fieldString(fields, "password"),
		APIKey:         fieldString(fields, "apiKey"),
		Category:       fieldString(fields, "movieCategory"),
	}
}
