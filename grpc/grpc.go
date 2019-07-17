// Common helper constructs for running a gRPC server.
package grpc

import (
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// An Option function can override configuration options
// for a server
type Option func(*Server)

// WithAddress overrides the default configured listen
// address for a server
func WithAddress(addr string) Option {
	return func(s *Server) {
		s.addr = addr
	}
}

// WithLogger overrides the logger instance
func WithLogger(log zerolog.Logger) Option {
	return func(s *Server) {
		s.log = log
	}
}

// WithServer overrides the grpc Server instance
func WithServer(srv *grpc.Server) Option {
	return func(s *Server) {
		s.srv = srv
	}
}

// RegisterServiceFunc registers a service with the gRPC server
// returning the service name
//
// Example:
//  var contentManager = func(srv *grpc.Server) string {
//  	pb.RegisterContentManagerServer(srv, &content.Manager{})
//  	return "kit.content.v1.ContentManager"
//  }
type RegisterServiceFunc func(*grpc.Server) string

// A Server can create and stop a gRPC server
//
// Example:
//  registerSvc := func(s *grpc.Server) string {
//  	healthpb.RegisterHealthServer(s, hs)
//  	return "kit.test.v1.Health"
//  }
//  s := grpc.New([]grpc.RegisterServiceFunc{registerSvc})
//  if err := s.Start(); err != nil {
//  	// handle server runtime err
//  }
//  if err := s.Stop(); err != nil {
//  	// handle server shutdown err
//  }
type Server struct {
	addr     string // address to bind too
	services []RegisterServiceFunc
	running  sync.Mutex // protects server running state
	srv      *grpc.Server
	log      zerolog.Logger
	errC     chan error
	sigC     chan os.Signal
}

// Start starts serving the gRPC server
func (s *Server) Start() error {
	s.running.Lock()
	defer s.running.Unlock()
	log := s.log.With().Str("func", "Server.Start").Logger()
	log.Debug().Str("listen", s.addr).Msg("opening net listener")
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	// Health check server
	hs := health.NewServer()
	// Register services
	for _, register := range s.services {
		serviceName := register(s.srv)
		hs.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)
	}
	// Register healthcheck server with gRPC server
	healthpb.RegisterHealthServer(s.srv, hs)
	// Start server
	log.Debug().Str("listen", s.addr).Msg("starting gRPC server")
	go func() { s.errC <- s.srv.Serve(listener) }()
	// Wait for OS signal or runtime error
	signal.Notify(s.sigC, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	select {
	case err := <-s.errC:
		return err
	case sig := <-s.sigC:
		log.Debug().Str("signal", sig.String()).Msg("received OS signal")
		return nil
	}
}

// Stop gracefully stops the grpc server
func (s *Server) Stop() error {
	s.running.Lock()
	defer s.running.Unlock()
	log := s.log.With().Str("func", "Server.Stop").Logger()
	if s.srv != nil {
		log.Debug().Msg("gracefully stopping gRPC server")
		s.srv.GracefulStop()
	}
	return nil

}

// New creates a new gRPC server. Provide a slice of service registers
// and use Option functions to override defaults.
func New(services []RegisterServiceFunc, opts ...Option) *Server {
	s := &Server{
		srv:      grpc.NewServer(),
		addr:     ":5000",
		log:      zerolog.New(os.Stdout),
		sigC:     make(chan os.Signal),
		errC:     make(chan error),
		services: services,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
