package otel

import (
	"context"

	"connectrpc.com/connect"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/stats"
)

// Creates a filter that simply calls every given filter, in the
// same order as they're passed, until either one returns false (to filter out the request)
// or until there are no filters left to check, thus allowing the request to be traced.
func FilterChain(filters ...otelgrpc.Filter) otelgrpc.Filter {
	return func(info *stats.RPCTagInfo) bool {
		for _, f := range filters {
			if !f(info) {
				return false
			}
		}
		return true
	}
}

// Creates a filter that prevents any methods listed from having a trace
// automatically be created.
//
// Note that methods should start with a slash and are in full form,
// e.g. `/grpc.health.v1.Health/Check`
//
// Note that method names must be an exact match to be filtered.
func FilterMethods(methods ...string) otelgrpc.Filter {
	// false = don't trace; true = trace
	return func(info *stats.RPCTagInfo) bool {
		if info == nil {
			return true
		}

		for _, filterMethod := range methods {
			if info.FullMethodName == filterMethod {
				return false
			}
		}

		return true
	}
}

type ConnectFilterFunc func(context.Context, connect.Spec) bool

// FilterMethodsConnect Creates a filter for otelconnect that prevents any methods listed from having a trace
// automatically be created.
//
// Note that methods should start with a slash and are in full form,
// e.g. `/grpc.health.v1.Health/Check`
//
// Note that method names must be an exact match to be filtered.
func FilterMethodsConnect(methods ...string) ConnectFilterFunc {
	// false = don't trace; true = trace
	return func(_ context.Context, spec connect.Spec) bool {
		for _, method := range methods {
			if spec.Procedure == method {
				return false
			}
		}
		return true
	}
}

// FilterMethodsConnectWithHealthCheck filters the given methods including the grpc healthcheck
func FilterMethodsConnectWithHealthCheck(additionalMethods ...string) ConnectFilterFunc {
	methods := append(additionalMethods, "/grpc.health.v1.Health/Check")
	return FilterMethodsConnect(methods...)
}
