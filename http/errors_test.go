package http_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	h "go.soon.build/kit/http"
)

func TestServer_Errs(t *testing.T) {
	s := h.New()
	tc := []struct {
		desc       string
		errHandler func(http.ResponseWriter, error, string)
		code       int
	}{
		{
			desc:       "ErrBadRequest",
			errHandler: s.ErrBadRequest,
			code:       http.StatusBadRequest,
		},
		{
			desc:       "ErrNotFound",
			errHandler: s.ErrNotFound,
			code:       http.StatusNotFound,
		},
		{
			desc:       "ErrInternal",
			errHandler: s.ErrInternal,
			code:       http.StatusInternalServerError,
		},
	}
	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {

			w := httptest.NewRecorder()
			errMsg := "response err msg"
			tc.errHandler(w, errors.New("unknown err"), errMsg)

			resp := w.Result()
			b, _ := ioutil.ReadAll(resp.Body)

			if resp.StatusCode != tc.code {
				t.Errorf("unexpected response status; expected %d, got %d", tc.code, resp.StatusCode)
			}
			if resp.Header.Get("Content-Type") != "application/json" {
				t.Error("unexpected Content-Type")
			}
			var body map[string]interface{}
			err := json.Unmarshal(b, &body)
			if err != nil {
				t.Fatal(err)
			}
			if body["message"] != errMsg {
				t.Errorf("unexpected body; expected %s, got %s", errMsg, body["message"])
			}
			code := int(body["code"].(float64))
			if code != tc.code {
				t.Errorf("unexpected body; expected %d, got %d", tc.code, code)
			}
		})
	}
}
