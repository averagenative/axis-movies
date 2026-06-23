package server

import (
	"crypto/subtle"
	"net/http"
)

// APIKey enforces Radarr-style API key auth. The key may be supplied via the
// X-Api-Key header or an apikey query parameter, matching Radarr clients.
func APIKey(want string) func(http.Handler) http.Handler {
	wantBytes := []byte(want)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("X-Api-Key")
			if got == "" {
				got = r.URL.Query().Get("apikey")
			}
			if subtle.ConstantTimeCompare([]byte(got), wantBytes) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
