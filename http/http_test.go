package http_test

import (
	"io/ioutil"
	"net/http"
	"syscall"
	"testing"
	"time"

	h "go.soon.build/kit/http"
)

func TestServer_StartStop(t *testing.T) {
	s := h.New()
	stopped := make(chan bool, 1)
	go func() {
		err := s.Start()
		if err != nil {
			t.Error(err)
		}
		stopped <- true
	}()
	time.Sleep(time.Second)
	if !s.Running {
		t.Errorf("server state is not running")
	}
	// test request
	res, err := http.Get("http://0.0.0.0:5000")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	expBody := `{"app":"kit","version":""}`
	if string(b) != expBody {
		t.Errorf("unexecpted body, expected %s, got %s", expBody, b)
	}

	// stop
	err = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	if err != nil {
		t.Fatal(err)
	}
	<-stopped
	err = s.Stop()
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)
	if s.Running {
		t.Errorf("server state is still running")
	}
}

func TestWithAddr(t *testing.T) {
	addr := ":9000"
	s := h.New(h.WithAddr(addr))
	if s.Srv.Addr != addr {
		t.Errorf("unexpected address; expected %s, got %s", addr, s.Srv.Addr)
	}
}

func TestWithHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	s := h.New(h.WithHandler(handler))
	if s.Srv.Handler == nil {
		t.Errorf("unexpected handler")
	}
}
