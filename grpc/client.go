package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// NewClient constructs a grpc client connection
func NewClient(server string, grpcOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	grpcOpts = append(grpcOpts,
		// For backwards compatibility, keep these previous, hardcoded options
		grpc.WithBlock(),
		grpc.WithInsecure(), // Note: Deprecated in newer versions for grpc.WithTransportCredentials(insecure.NewCredentials())
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	return grpc.DialContext(
		ctx,
		server,
		grpcOpts...,
	)
}
