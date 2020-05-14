package http

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

// Health returns a handler for healthcheck requests
func (s *Server) Health(app, version string) http.Handler {
	type healthResponse struct {
		App     string `json:"app"`
		Version string `json:"version"`
		Serving bool   `json:"serving"`
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := zerolog.Ctx(r.Context())
		if log == nil {
			log = &s.log
		}
		b, _ := json.Marshal(healthResponse{
			App:     app,
			Version: version,
			Serving: s.Running,
		})
		w.Header().Set("Content-Type", "application/json")
		if s.Running {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_, err := w.Write(b)
		if err != nil {
			log.Error().Err(err).Msg("error writing to response")
		}
	})
}
