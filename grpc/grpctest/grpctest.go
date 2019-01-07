package grpctest

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// RegisterServiceFunc registers a service with the test gRPC server
type RegisterServiceFunc func(*grpc.Server)

// ServerClient creates a gRPC server and client connection for the server
// Usage:
//  srv, cc := grpctest.ServerClient(t, func(srv *grpc.Server) { ...  })
//  defer cc.Close()
//  defer srv.GracefulStop()
//  client := pb.NewArticleManagerClient(cc)
func ServerClient(t *testing.T, services ...RegisterServiceFunc) (*grpc.Server, *grpc.ClientConn) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.FailNow()
	}
	srv := grpc.NewServer()
	for _, register := range services {
		register(srv)
	}
	go func() {
		t.Logf("started gRPC server on %s", ln.Addr().String())
		if err := srv.Serve(ln); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cc, err := grpc.DialContext(
		ctx,
		ln.Addr().String(),
		grpc.WithBlock(),
		grpc.WithInsecure())
	if err != nil {
		t.FailNow()
	}
	return srv, cc
}
