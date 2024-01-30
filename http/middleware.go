package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// Middleware represents a func that chains http handlers
type Middleware func(next http.Handler) http.Handler

// Used with logging handlers. Returns true if the request should NOT be logged.
type LogFilter func(r *http.Request) bool

// DefaultRequestLogger provides a default middleware chain with
// AccessHandler and RequestIDHandler middlewares
//
// Example:
//
//	DefaultRequestLogger(log, "requestid", "Request-Id")(handler)
var DefaultRequestLogger = func(log zerolog.Logger, fieldKey, headerName string) Middleware {
	return func(next http.Handler) http.Handler {
		return hlog.NewHandler(log)(
			AccessHandler(
				RequestIDHandler(fieldKey, headerName)(next),
				func(r *http.Request) bool {
					return r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/__")
				},
			),
		)
	}
}

// AccessHandler is a standard request logger implementation
//
// Any path that contains a prefix from excludedPathPrefixes will not be logged.
// This is useful for preventing health checks from being logged out.
func AccessHandler(next http.Handler, filters ...LogFilter) http.Handler {
	handler := hlog.AccessHandler(func(r *http.Request, status, size int, dur time.Duration) {
		for _, filter := range filters {
			if filter(r) {
				return
			}
		}

		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", status).
			Int("size", size).
			Dur("duration", dur).
			Msg("handled http request")
	})
	return handler(next)
}

type idKey struct{}

// IDFromRequest returns the unique id associated with the request. This is
// retrieved from the context or a header on the incoming request if available.
func IDFromRequest(r *http.Request, headerName string) (string, bool) {
	if r == nil {
		return "", false
	}
	id, ok := IDFromCtx(r.Context())
	if ok {
		return id, ok
	}
	if headerName != "" {
		id = r.Header.Get(headerName)
		if id == "" {
			return id, false
		}
	}
	return id, true
}

// IDFromCtx returns the unique id associated to the context if any.
func IDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(idKey{}).(string)
	return id, ok
}

// RequestIDHandler returns a handler setting a unique id on the request which can
// be retrieved using IDFromRequest(req). This generated id is added as a field to the
// logger using the passed fieldKey as field name. The id is also added as a response
// header if the headerName is not empty.
func RequestIDHandler(fieldKey, headerName string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			id, ok := IDFromRequest(r, headerName)
			if !ok {
				id = xid.New().String()
				ctx = context.WithValue(ctx, idKey{}, id)
				r = r.WithContext(ctx)
			}
			if fieldKey != "" {
				log := zerolog.Ctx(ctx)
				log.UpdateContext(func(c zerolog.Context) zerolog.Context {
					return c.Str(fieldKey, id)
				})
			}
			if headerName != "" {
				w.Header().Set(headerName, id)
			}
			next.ServeHTTP(w, r)
		})
	}
}
