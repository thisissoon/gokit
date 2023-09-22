package grpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	g "go.soon.build/kit/grpc"
)

func TestRequestID(t *testing.T) {
	testCases := []struct {
		desc      string
		context   context.Context
		fieldName string
		expIDLen  int
	}{
		{
			desc: "id from context",
			context: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"requestid": []string{"123"},
			}),
			fieldName: "requestid",
			expIDLen:  3,
		},
		{
			desc:      "new id, no metadata",
			context:   context.Background(),
			fieldName: "requestid",
			expIDLen:  20,
		},
		{
			desc:      "new id, no request id field",
			context:   metadata.NewIncomingContext(context.Background(), metadata.MD{}),
			fieldName: "requestid",
			expIDLen:  20,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			id := g.RequestID(tc.context, tc.fieldName)
			if len(id) != tc.expIDLen {
				t.Errorf("unexpected ID length; expected %d, got %d", tc.expIDLen, len(id))
			}
		})
	}
}

func TestTraceID(t *testing.T) {
	testCases := []struct {
		desc      string
		context   context.Context
		fieldName string
		want      string
	}{
		{
			desc: "trace id from context",
			context: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"traceid": []string{"123"},
			}),
			fieldName: "traceid",
			want:      "123",
		},
		{
			desc:      "new id, no metadata",
			context:   context.Background(),
			fieldName: "traceid",
			want:      "",
		},
		{
			desc:      "new id, no trace id field",
			context:   metadata.NewIncomingContext(context.Background(), metadata.MD{}),
			fieldName: "traceid",
			want:      "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			id := g.TraceID(tc.context, tc.fieldName)
			if id != tc.want {
				t.Errorf("mismatching ID's; expected %s, got %s", tc.want, id)
			}
		})
	}
}

func TestLogUnaryInterceptor(t *testing.T) {
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		log := zerolog.Ctx(ctx)
		if log == nil {
			t.Error("log not attached to context")
		}
		if req.(string) != "request" {
			t.Error("unexpected request arg")
		}
		return nil, nil
	}
	tests := map[string]struct {
		traceField             g.TraceField
		requestTraceField      string
		requestTraceFieldValue string
		xTraceLoggingField     string
	}{
		"use the default traceField values for an empty traceField": {
			traceField:             g.TraceField{},
			requestTraceField:      "x-b3-traceid",
			requestTraceFieldValue: "trace-id",
			xTraceLoggingField:     "logging.googleapis.com/trace",
		},
		"override the trace field with given values": {
			traceField: g.TraceField{
				LoggingFieldName: "logging/traceid",
				RequestFieldName: "traceid",
			},
			requestTraceField:      "traceid",
			requestTraceFieldValue: "trace-id",
			xTraceLoggingField:     "logging/traceid",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			logWriter := bytes.Buffer{}
			interceptor := g.LogUnaryInterceptor(
				zerolog.New(&logWriter),
				"requestid",
				tt.traceField,
			)
			// set trace id to the context
			md := metadata.New(map[string]string{
				tt.requestTraceField: tt.requestTraceFieldValue,
			})
			ctx := metadata.NewIncomingContext(context.Background(), md)
			_, err := interceptor(
				ctx,
				"request",
				&grpc.UnaryServerInfo{FullMethod: "list"},
				handler,
			)
			if err != nil {
				t.Fatal(err)
			}
			entries := logEntriesFromBuffer(t, logWriter)
			assert.Equal(t, "list", entries[0]["grpc.method"], "unexpected grpc method")
			assert.Equal(t, "handled gRPC unary request", entries[0]["message"], "unexpected log message")
			assert.NotNil(t, entries[0]["grpc.duration"], "missing grpc.duration field")
			assert.NotNil(t, entries[0]["requestid"], "missing requestid field")
			assert.NotNil(t, entries[0][tt.xTraceLoggingField], "missing trace log field")
			assert.Equal(t, tt.requestTraceFieldValue, entries[0][tt.xTraceLoggingField])
		})
	}
}

func TestLogStreamInterceptor(t *testing.T) {
	handler := func(srv interface{}, ws grpc.ServerStream) error {
		log := zerolog.Ctx(ws.Context())
		if log == nil {
			t.Error("log not attached to context")
		}
		if srv.(string) != "request" {
			t.Error("unexpected request arg")
		}
		return nil
	}
	tests := map[string]struct {
		traceField             g.TraceField
		requestTraceField      string
		requestTraceFieldValue string
		xTraceLoggingField     string
	}{
		"use the default traceField values for an empty traceField": {
			traceField:             g.TraceField{},
			requestTraceField:      "x-b3-traceid",
			requestTraceFieldValue: "trace-id",
			xTraceLoggingField:     "logging.googleapis.com/trace",
		},
		"override the trace field with given values": {
			traceField: g.TraceField{
				LoggingFieldName: "logging/traceid",
				RequestFieldName: "traceid",
			},
			requestTraceField:      "traceid",
			requestTraceFieldValue: "trace-id",
			xTraceLoggingField:     "logging/traceid",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			logWriter := bytes.Buffer{}
			interceptor := g.LogStreamInterceptor(
				zerolog.New(&logWriter),
				"requestid",
				tt.traceField,
			)
			// set trace id to the context
			md := metadata.New(map[string]string{
				tt.requestTraceField: tt.requestTraceFieldValue,
			})
			ctx := metadata.NewIncomingContext(context.Background(), md)
			err := interceptor(
				"request",
				&g.WrappedServerStream{WrappedContext: ctx},
				&grpc.StreamServerInfo{FullMethod: "list"},
				handler,
			)
			if err != nil {
				t.Fatal(err)
			}
			entries := logEntriesFromBuffer(t, logWriter)
			assert.Equal(t, "list", entries[0]["grpc.method"], "unexpected grpc method")
			assert.Equal(t, "handled gRPC stream request", entries[0]["message"], "unexpected log message")
			assert.NotNil(t, entries[0]["grpc.duration"], "missing grpc.duration field")
			assert.NotNil(t, entries[0]["requestid"], "missing requestid field")
			assert.NotNil(t, entries[0][tt.xTraceLoggingField], "missing trace log field")
			assert.Equal(t, tt.requestTraceFieldValue, entries[0][tt.xTraceLoggingField])
		})
	}
}

func logEntriesFromBuffer(t *testing.T, buff bytes.Buffer) []map[string]interface{} {
	parts := strings.Split(buff.String(), "\n")
	entries := make([]map[string]interface{}, len(parts))
	for i, e := range parts {
		entries[i] = map[string]interface{}{}
		if e != "" {
			err := json.Unmarshal([]byte(e), &entries[i])
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	return entries
}
