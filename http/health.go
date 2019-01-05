package http

import (
	"encoding/json"
	"net/http"
)

// Health returns a handler for healthcheck requests
func (s *Server) Health(app, version string) http.Handler {
	type healthResponse struct {
		App     string `json:"app"`
		Version string `json:"version"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(healthResponse{
			App:     app,
			Version: version,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(b)
		if err != nil {
			s.log.Error().Err(err).Msg("error writing to response")
		}
	})
}
