package grpc_test

import (
	"errors"
	"net"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"

	grpckit "go.soon.build/kit/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestHealthRequest(t *testing.T) {
	testCases := []struct {
		desc string
		svc  []string
		err  error
	}{
		{
			desc: "success",
			svc:  []string{"test"},
		},
		{
			desc: "fail",
			svc:  []string{"test", "unknown"},
			err:  errors.New("could not create gRPC client connection: rpc error: code = NotFound desc = unknown service"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			host := ":50000"
			listener, err := net.Listen("tcp", host)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			srv := grpc.NewServer()
			defer srv.Stop()
			hs := health.NewServer()
			hs.SetServingStatus("test", healthpb.HealthCheckResponse_SERVING)
			healthpb.RegisterHealthServer(srv, hs)
			go func() {
				_ = srv.Serve(listener)
			}()
			err = grpckit.HealthRequest(host, tc.svc, "requestID", zerolog.New(os.Stdout))
			if tc.err == nil && err != nil {
				t.Fatal(err)
			}
			if tc.err != nil && err.Error() != tc.err.Error() {
				t.Errorf("unexpected err; expected %v, got %v", tc.err, err)
			}
		})
	}
}
