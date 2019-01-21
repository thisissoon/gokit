package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/metadata"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// HealthRequest makes a healthcheck request to gRPC service
func HealthRequest(host string, svc []string, reqIDField string, log zerolog.Logger) error {
	cc, err := NewClient(host)
	if err != nil {
		return fmt.Errorf("cound not create gRPC client connection: %v", err)
	}
	hc := healthpb.NewHealthClient(cc)
	var wg sync.WaitGroup
	errC := make(chan error, 1)
	for _, svc := range svc {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()
			requestID := xid.New().String()
			l := log.With().Str(reqIDField, requestID).Logger()
			md := metadata.Pairs(reqIDField, requestID)
			ctx := metadata.NewOutgoingContext(context.Background(), md)
			l.Debug().Msg("running healthceck")
			rsp, err := hc.Check(ctx, &healthpb.HealthCheckRequest{
				Service: svc,
			})
			switch err {
			case nil:
				l.Debug().
					Str("service", svc).
					Str("status", rsp.GetStatus().String()).
					Msg("service status")
				switch rsp.GetStatus() {
				case healthpb.HealthCheckResponse_NOT_SERVING, healthpb.HealthCheckResponse_UNKNOWN:
					errC <- fmt.Errorf("%s: not serving", svc)
				}
			default:
				errC <- fmt.Errorf("cound not create gRPC client connection: %v", err)
			}
		}(svc)
	}
	wg.Wait()
	close(errC)
	err = <-errC
	return err
}
