package otel

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// Creates a filter that simply calls every given filter, in the
// same order as they're passed, until either one returns false (to filter out the request)
// or until there are no filters left to check, thus allowing the request to be traced.
func FilterChain(filters ...otelgrpc.Filter) otelgrpc.Filter {
	return func(ii *otelgrpc.InterceptorInfo) bool {
		for _, f := range filters {
			if !f(ii) {
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
	return func(ii *otelgrpc.InterceptorInfo) bool {
		if ii == nil {
			return true
		}

		var method string
		if ii.StreamServerInfo != nil && len(ii.StreamServerInfo.FullMethod) > 0 {
			method = ii.StreamServerInfo.FullMethod
		} else if ii.UnaryServerInfo != nil {
			method = ii.UnaryServerInfo.FullMethod
		} else {
			method = ii.Method
		}

		for _, filterMethod := range methods {
			if method == filterMethod {
				return false
			}
		}

		return true
	}
}
