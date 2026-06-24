package v3

import "net/http"

func (a *API) serverError(w http.ResponseWriter, err error) {
	a.log.Error("api error", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

// --- movies (read-only in Phase 1; populated via TMDb add in Phase 2) ---

func (a *API) listMovies(w http.ResponseWriter, r *http.Request) {
	rows, err := a.q.ListMovies(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, m := range rows {
		out = append(out, movieJSON(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getMovie(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	m, err := a.q.GetMovie(r.Context(), id)
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "movie not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, movieJSON(m))
	}
}

// --- root folders ---

func (a *API) listRootFolders(w http.ResponseWriter, r *http.Request) {
	rows, err := a.q.ListRootFolders(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, rf := range rows {
		out = append(out, rootFolderJSON(rf))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getRootFolder(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rf, err := a.q.GetRootFolder(r.Context(), id)
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "root folder not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, rootFolderJSON(rf))
	}
}

func (a *API) createRootFolder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	rf, err := a.q.CreateRootFolder(r.Context(), body.Path)
	switch {
	case isUniqueViolation(err):
		writeError(w, http.StatusConflict, "root folder already exists")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusCreated, rootFolderJSON(rf))
	}
}

func (a *API) deleteRootFolder(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := a.q.DeleteRootFolder(r.Context(), id); err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// --- tags ---

func (a *API) listTags(w http.ResponseWriter, r *http.Request) {
	rows, err := a.q.ListTags(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, t := range rows {
		out = append(out, tagJSON(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getTag(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := a.q.GetTag(r.Context(), id)
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "tag not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, tagJSON(t))
	}
}

func (a *API) createTag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Label string `json:"label"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Label == "" {
		writeError(w, http.StatusBadRequest, "label is required")
		return
	}
	t, err := a.q.CreateTag(r.Context(), body.Label)
	switch {
	case isUniqueViolation(err):
		writeError(w, http.StatusConflict, "tag already exists")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusCreated, tagJSON(t))
	}
}

func (a *API) deleteTag(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := a.q.DeleteTag(r.Context(), id); err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// --- quality profiles (read-only in Phase 1; full engine in Phase 4) ---

func (a *API) listQualityProfiles(w http.ResponseWriter, r *http.Request) {
	rows, err := a.q.ListQualityProfiles(r.Context())
	if err != nil {
		a.serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, qp := range rows {
		out = append(out, qualityProfileJSON(qp))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) getQualityProfile(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	qp, err := a.q.GetQualityProfile(r.Context(), id)
	switch {
	case isNotFound(err):
		writeError(w, http.StatusNotFound, "quality profile not found")
	case err != nil:
		a.serverError(w, err)
	default:
		writeJSON(w, http.StatusOK, qualityProfileJSON(qp))
	}
}
