// Provides code for managing http server lifecycle. Also includes some
// helpers for error handling and common middleware.
package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
)

// Server manages the lifecycle of an http API server with graceful shutdown
//
// Example:
// 	srv := http.New()
// 	if err := srv.Start(); err != nil {
// 		// handle server runtime err
// 	}
// 	if err := s.Stop(); err != nil {
// 		// handle server close err
// 	}
type Server struct {
	Srv     *http.Server
	Running bool
	log     zerolog.Logger
}

// New constructs a server
func New(opts ...Option) *Server {
	s := &Server{
		Srv: &http.Server{Addr: ":5000"},
		log: zerolog.New(os.Stdout),
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.Srv.Handler == nil {
		s.Srv.Handler = s.Health("kit", "")
	}
	return s
}

// Option configures a Server instance
type Option func(*Server)

// WithAddr returns an Option to configure server listen address
func WithAddr(addr string) Option {
	return func(s *Server) {
		s.Srv.Addr = addr
	}
}

// WithLogger returns an Option to configure server logger instance
func WithLogger(l zerolog.Logger) Option {
	return func(s *Server) {
		s.log = l
	}
}

// WithHandler returns an Option to configure the server's http handler
func WithHandler(h http.Handler) Option {
	return func(s *Server) {
		s.Srv.Handler = h
	}
}

// Start starts the server listening, will block on signal or error
func (s *Server) Start() error {
	errC := make(chan error, 1)
	// listen
	go func() {
		s.log.Debug().Msg(fmt.Sprintf("listening on %s", s.Srv.Addr))
		s.Running = true
		err := s.Srv.ListenAndServe()
		switch err {
		case http.ErrServerClosed:
			s.Running = false
			s.log.Debug().Err(err).Bool("running", s.Running).Msg("server closed")
		case nil:
		default:
			errC <- err
		}
		close(errC)
	}()

	// wait for OS signal or runtime error
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	select {
	case err := <-errC:
		return err
	case sig := <-stop:
		s.log.Debug().Str("signal", sig.String()).Msg("signal received")
		return nil
	}
}

// Stop stops the running server
func (s *Server) Stop() error {
	if s.Srv != nil {
		s.log.Debug().Msg("gracefully stopping server")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		return s.Srv.Shutdown(ctx)
	}
	return nil
}
