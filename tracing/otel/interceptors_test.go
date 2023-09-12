package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func TestFilterMethods(t *testing.T) {
	cases := map[string]struct {
		info    *otelgrpc.InterceptorInfo
		methods []string
		expect  bool
	}{
		"empty should default to being traced": {
			info:    &otelgrpc.InterceptorInfo{},
			methods: []string{},
			expect:  true,
		},
		"matches on StreamServerInfo should work": {
			info: &otelgrpc.InterceptorInfo{
				StreamServerInfo: &grpc.StreamServerInfo{
					FullMethod: "/abc.Service/Method",
				},
			},
			methods: []string{"/abc.Service/Method"},
			expect:  false,
		},
		"matches on UnaryServerInfo should work": {
			info: &otelgrpc.InterceptorInfo{
				UnaryServerInfo: &grpc.UnaryServerInfo{
					FullMethod: "/abc.Service/Method",
				},
			},
			methods: []string{"/abc.Service/Method"},
			expect:  false,
		},
		"matches on Method should work": {
			info: &otelgrpc.InterceptorInfo{
				Method: "/abc.Service/Method",
			},
			methods: []string{"/abc.Service/Method"},
			expect:  false,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			filter := FilterMethods(testCase.methods...)
			result := filter(testCase.info)
			assert.Equal(t, testCase.expect, result)
		})
	}
}

func TestFilterChain(t *testing.T) {
	cases := map[string]struct {
		info    *otelgrpc.InterceptorInfo
		filters []otelgrpc.Filter
		expect  bool
	}{
		"defaults to accepting the trace": {
			info:    &otelgrpc.InterceptorInfo{},
			filters: []otelgrpc.Filter{},
			expect:  true,
		},
		"blocks the trace if any filter blocks the trace": {
			info: &otelgrpc.InterceptorInfo{},
			filters: []otelgrpc.Filter{
				func(ii *otelgrpc.InterceptorInfo) bool { return false },
			},
			expect: false,
		},
		"accepts the trace if no filters block it": {
			info: &otelgrpc.InterceptorInfo{},
			filters: []otelgrpc.Filter{
				func(ii *otelgrpc.InterceptorInfo) bool { return true },
			},
			expect: true,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			filter := FilterChain(testCase.filters...)
			result := filter(testCase.info)
			assert.Equal(t, testCase.expect, result)
		})
	}
}
