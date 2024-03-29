// Package http provides code for managing http server lifecycle. Also includes some
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
// 	if err := srv.Start(ctx); err != nil {
// 		// handle server runtime err
// 	}
// 	if err := s.Stop(); err != nil {
// 		// handle server close err
// 	}
type Server struct {
	Srv         *http.Server
	Running     bool
	log         zerolog.Logger
	stopTimeout time.Duration
	handler     http.Handler
	healthOpt   HealthOptions
}

// New constructs a server
func New(opts ...Option) *Server {
	s := &Server{
		Srv:         &http.Server{Addr: ":5000"},
		log:         zerolog.New(os.Stdout),
		stopTimeout: time.Second * 10,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.Srv.Handler = s.handler
	if s.handler == nil {
		s.Srv.Handler = s.Health(HealthOptions{AppName: "kit"})
	}
	if s.healthOpt.Path != "" {
		mux := http.NewServeMux()
		mux.Handle("/", s.handler)
		mux.Handle(s.healthOpt.Path, s.Health(s.healthOpt))
		s.Srv.Handler = mux
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
		s.handler = h
	}
}

// WithHealth returns an Option to configure the server healthcheck endpoint
func WithHealth(h HealthOptions) Option {
	return func(s *Server) {
		s.healthOpt = h
	}
}

// WithStopTimeout returns an Option to configure the duration
// to wait for connections to terminate on shutdown
func WithStopTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.stopTimeout = d
	}
}

// Start starts the server listening, will block on signal or error
func (s *Server) Start(ctx context.Context) error {
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

	// wait for ctx done or runtime error
	select {
	case err := <-errC:
		return err
	case <-ctx.Done():
		return s.Stop()
	}
}

// Stop stops the running server
func (s *Server) Stop() error {
	if s.Srv != nil {
		s.log.Debug().Msg("gracefully stopping server")
		ctx, cancel := context.WithTimeout(context.Background(), s.stopTimeout)
		defer cancel()
		return s.Srv.Shutdown(ctx)
	}
	return nil
}

// CtxWithSignal returns a context that completes when one of the
// os Signals is received. Leaving sig empty will default to SIGTERM, SIGQUIT and SIGINT
func CtxWithSignal(ctx context.Context, sig ...os.Signal) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	stop := make(chan os.Signal, 1)
	if len(sig) < 1 {
		sig = []os.Signal{syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT}
	}
	signal.Notify(stop, sig...)
	go func() {
		<-stop
		cancel()
		signal.Stop(stop)
	}()
	return ctx
}
