package v3

import (
	"encoding/json"
	"time"

	"github.com/averagenative/axis-movies/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- pgtype -> plain value helpers ---

func textVal(v pgtype.Text) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func int4Val(v pgtype.Int4) int32 {
	if v.Valid {
		return v.Int32
	}
	return 0
}

func int8Val(v pgtype.Int8) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

func tsVal(v pgtype.Timestamptz) any {
	if v.Valid {
		return v.Time.UTC().Format(time.RFC3339)
	}
	return nil
}

// rawJSON returns stored JSONB bytes as raw JSON, defaulting to an empty array.
func rawJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return json.RawMessage("[]")
	}
	return json.RawMessage(b)
}

// --- store model -> Radarr v3 resource shapes ---

func rootFolderJSON(rf store.RootFolder) map[string]any {
	return map[string]any{
		"id":              rf.ID,
		"path":            rf.Path,
		"accessible":      true,
		"freeSpace":       0,
		"unmappedFolders": []any{},
	}
}

func tagJSON(t store.Tag) map[string]any {
	return map[string]any{
		"id":    t.ID,
		"label": t.Label,
	}
}

func qualityProfileJSON(qp store.QualityProfile) map[string]any {
	return map[string]any{
		"id":             qp.ID,
		"name":           qp.Name,
		"upgradeAllowed": qp.UpgradeAllowed,
		"cutoff":         int4Val(qp.CutoffQualityID),
		"items":          rawJSON(qp.Items),
	}
}

func movieJSON(m store.Movie) map[string]any {
	return map[string]any{
		"id":               m.ID,
		"title":            m.Title,
		"tmdbId":           m.TmdbID,
		"year":             int4Val(m.Year),
		"monitored":        m.Monitored,
		"hasFile":          m.HasFile,
		"runtime":          m.Runtime,
		"titleSlug":        textVal(m.TitleSlug),
		"sortTitle":        textVal(m.SortTitle),
		"overview":         textVal(m.Overview),
		"status":           textVal(m.Status),
		"imdbId":           textVal(m.ImdbID),
		"path":             textVal(m.Path),
		"rootFolderPath":   textVal(m.RootFolderPath),
		"qualityProfileId": int8Val(m.QualityProfileID),
		"images":           rawJSON(m.Images),
		"added":            tsVal(m.AddedAt),
	}
}
