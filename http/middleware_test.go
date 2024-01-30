package http_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	h "go.soon.build/kit/http"
)

func TestRequestIDHandler(t *testing.T) {
	tc := []struct {
		desc  string
		req   func() *http.Request
		expID string
	}{
		{
			desc: "existing id",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", "http://example.com/foo", nil)
				req.Header.Set("Request-ID", "existing")
				return req
			},
			expID: "existing",
		},
		{
			desc: "new request id",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", "http://example.com/foo", nil)
				return req
			},
		},
	}
	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log := zerolog.Ctx(r.Context())
				log.Debug().Msg("handler debug msg")
				_, err := io.WriteString(w, "<html><body>Hello World!</body></html>")
				if err != nil {
					t.Fatal(err)
				}
			})

			w := httptest.NewRecorder()
			logWriter := bytes.Buffer{}
			log := zerolog.New(&logWriter)
			chain := hlog.NewHandler(log)(h.RequestIDHandler("requestid", "Request-ID")(handler))
			chain.ServeHTTP(w, tc.req())

			// assertions
			entries := logEntriesFromBuffer(logWriter)
			if entries[0]["message"] != "handler debug msg" {
				t.Errorf("unexpected log msg; expected %s, got %s",
					"handler debug msg",
					entries[0]["message"])
			}
			if entries[0]["requestid"] == "" {
				t.Errorf("missing log field; requestid")
			}
			if tc.expID != "" {
				if entries[0]["requestid"] != tc.expID {
					t.Errorf("unexpected value for requestid; expected %s, got %s",
						tc.expID,
						entries[0]["requestid"])
				}
			}
		})
	}
}

func TestAccessHandler(t *testing.T) {
	tc := []struct {
		desc string
	}{
		{
			desc: "",
		},
	}
	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log := hlog.FromRequest(r)
				log.Debug().Msg("handler debug msg")
				_, err := io.WriteString(w, "<html><body>Hello World!</body></html>")
				if err != nil {
					t.Fatal(err)
				}
			})

			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			w := httptest.NewRecorder()
			logWriter := bytes.Buffer{}
			log := zerolog.New(&logWriter)
			chain := hlog.NewHandler(log)(h.AccessHandler(handler))
			chain.ServeHTTP(w, req)
			entries := logEntriesFromBuffer(logWriter)
			if entries[0]["message"] != "handler debug msg" {
				t.Errorf("unexpected log msg; expected %s, got %s", "handler debug msg", entries[0]["message"])
			}
			if entries[1]["message"] != "handled http request" {
				t.Errorf("unexpected log msg; expected %s, got %s", "handled http request", entries[1]["message"])
			}
			if entries[1]["method"] != "GET" {
				t.Errorf("unexpected log field; expected %s, got %s", "GET", entries[1]["method"])
			}
			if entries[1]["url"] != "http://example.com/foo" {
				t.Errorf("unexpected log field; expected %s, got %s", "http://example.com/foo", entries[1]["url"])
			}
		})
	}
}

func TestAccessHandlerFilter(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.WriteString(w, "<html><body>Hello World!</body></html>")
		if err != nil {
			t.Fatal(err)
		}
	})

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	logWriter := bytes.Buffer{}
	log := zerolog.New(&logWriter)
	chain := hlog.NewHandler(log)(h.AccessHandler(handler, func(r *http.Request) bool { return true }))
	chain.ServeHTTP(w, req)
	entries := logEntriesFromBuffer(logWriter)
	if len(entries) != 0 {
		t.Errorf("unexpected log entries; expected 0 entries, but got %d instead.", len(entries))
	}
}

func logEntriesFromBuffer(buff bytes.Buffer) []map[string]interface{} {
	parts := strings.Split(buff.String(), "\n")
	var entries []map[string]interface{}
	for i, e := range parts {
		if e == "" {
			continue
		}
		entries = append(entries, map[string]interface{}{})
		err := json.Unmarshal([]byte(e), &entries[i])
		if err != nil {
			log.Print(err)
		}
	}
	return entries
}
