package grpc_test

import (
	"context"
	"syscall"
	"testing"

	g "go.soon.build/kit/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestServer_StartStop(t *testing.T) {
	s := g.New([]g.RegisterServiceFunc{})
	stopped := make(chan bool, 1)
	go func() {
		err := s.Start()
		if err != nil {
			t.Error(err)
		}
		stopped <- true
	}()
	// test health method
	cc, err := g.NewClient(":5000")
	if err != nil {
		t.Fatal(err)
	}
	client := healthpb.NewHealthClient(cc)
	res, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != healthpb.HealthCheckResponse_SERVING {
		t.Error("server healthcheck failed")
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
}
