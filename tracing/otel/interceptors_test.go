package otel

import (
	"context"
	"testing"

	"connectrpc.com/connect"
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

type method string

func (method method) ConnectMethod() connect.Spec {
	if string(method) == "" {
		return connect.Spec{}
	}
	return connect.Spec{
		Procedure: string(method),
	}
}

func (method method) GRPCMethod() *stats.RPCTagInfo {
	if string(method) == "" {
		return &stats.RPCTagInfo{}
	}
	return &stats.RPCTagInfo{
		FullMethodName: string(method),
	}
}

type filterMethodTestCase map[string]struct {
	method  method
	methods []string
	expect  bool
}

var filterMethodTests = filterMethodTestCase{
	"empty should default to being traced": {
		method:  method(""),
		methods: []string{},
		expect:  true,
	},
	"matches on StreamServerInfo should work": {
		method:  method("/abc.Service/Method"),
		methods: []string{"/abc.Service/Method"},
		expect:  false,
	},
	"matches on Method should work": {
		method:  method("/abc.Service/Method"),
		methods: []string{"/abc.Service/Method"},
		expect:  false,
	},
}

func TestFilterMethods(t *testing.T) {
	for name, testCase := range filterMethodTests {
		t.Run(name, func(t *testing.T) {
			filter := FilterMethods(testCase.methods...)
			result := filter(testCase.method.GRPCMethod())
			assert.Equal(t, testCase.expect, result)
		})
	}
}

func TestFilterMethodsConnect(t *testing.T) {
	for name, testCase := range filterMethodTests {
		t.Run(name, func(t *testing.T) {
			filter := FilterMethodsConnect(testCase.methods...)
			result := filter(context.Background(), testCase.method.ConnectMethod())
			assert.Equal(t, testCase.expect, result)
		})
	}
}

func TestFilterMethodsConnectWithHealthCheck(t *testing.T) {
	tests := map[string]struct {
		method            method
		additionalMethods []string
		expect            bool
	}{
		"filter healthcheck - no additional methods": {
			method:            "/grpc.health.v1.Health/Check",
			additionalMethods: []string{},
			expect:            false,
		},
		"filter healthcheck - with additional methods": {
			method:            "/grpc.health.v1.Health/Check",
			additionalMethods: []string{"/abc.Service/Method"},
			expect:            false,
		},
		"filter additional method": {
			method:            "/abc.Service/Method",
			additionalMethods: []string{"/abc.Service/Method"},
			expect:            false,
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			filter := FilterMethodsConnectWithHealthCheck(testCase.additionalMethods...)
			result := filter(context.Background(), testCase.method.ConnectMethod())
			assert.Equal(t, testCase.expect, result)
		})
	}
}
