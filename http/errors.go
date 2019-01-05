package http

import (
	"encoding/json"
	"net/http"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
)

type errResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	ErrID   string `json:"errID"`
}

// ErrBadRequest writes a bad request err response
func (s *Server) ErrBadRequest(w http.ResponseWriter, err error, msg string) {
	RequestErr(http.StatusBadRequest, s.log, w, err, msg)
}

// ErrInternal writes an internal err response
func (s *Server) ErrInternal(w http.ResponseWriter, err error, msg string) {
	RequestErr(http.StatusInternalServerError, s.log, w, err, msg)
}

// ErrNotFound writes a not found err response
func (s *Server) ErrNotFound(w http.ResponseWriter, err error, msg string) {
	RequestErr(http.StatusNotFound, s.log, w, err, msg)
}

// RequestErr handles logs an error and writes an error response
func RequestErr(status int, log zerolog.Logger, w http.ResponseWriter, err error, msg string) {
	errID := makeErrID()
	log.Error().Str("errID", errID).Err(err).Msg(msg)
	res := errResponse{
		Message: msg,
		Code:    status,
		ErrID:   errID,
	}
	b, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(b)
	if err != nil {
		log.Error().Err(err).Msg("error writing to response")
	}
}

var makeErrID = func() string {
	return xid.New().String()
}
