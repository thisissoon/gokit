package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// NewClient constructs a grpc client connection
func NewClient(server string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	return grpc.DialContext(
		ctx,
		server,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
}
