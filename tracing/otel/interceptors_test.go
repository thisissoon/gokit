package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/stats"
)

func TestFilterChain(t *testing.T) {
	cases := map[string]struct {
		info    *stats.RPCTagInfo
		filters []otelgrpc.Filter
		expect  bool
	}{
		"defaults to accepting the trace": {
			info:    &stats.RPCTagInfo{},
			filters: []otelgrpc.Filter{},
			expect:  true,
		},
		"blocks the trace if any filter blocks the trace": {
			info: &stats.RPCTagInfo{},
			filters: []otelgrpc.Filter{
				func(ii *stats.RPCTagInfo) bool { return false },
			},
			expect: false,
		},
		"accepts the trace if no filters block it": {
			info: &stats.RPCTagInfo{},
			filters: []otelgrpc.Filter{
				func(ii *stats.RPCTagInfo) bool { return true },
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

func TestFilterMethods(t *testing.T) {
	cases := map[string]struct {
		info    *stats.RPCTagInfo
		methods []string
		expect  bool
	}{
		"empty should default to being traced": {
			info:    &stats.RPCTagInfo{},
			methods: []string{},
			expect:  true,
		},
		"matches on StreamServerInfo should work": {
			info: &stats.RPCTagInfo{
				FullMethodName: "/abc.Service/Method",
			},
			methods: []string{"/abc.Service/Method"},
			expect:  false,
		},
		"matches on UnaryServerInfo should work": {
			info: &stats.RPCTagInfo{
				FullMethodName: "/abc.Service/Method",
			},
			methods: []string{"/abc.Service/Method"},
			expect:  false,
		},
		"matches on Method should work": {
			info: &stats.RPCTagInfo{
				FullMethodName: "/abc.Service/Method",
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
