package grpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"

	"github.com/rs/zerolog"
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
	logWriter := bytes.Buffer{}
	interceptor := g.LogUnaryInterceptor(
		zerolog.New(&logWriter),
		"requestid",
	)
	_, err := interceptor(
		context.Background(),
		"request",
		&grpc.UnaryServerInfo{FullMethod: "list"},
		handler,
	)
	if err != nil {
		t.Fatal(err)
	}
	entries := logEntriesFromBuffer(logWriter)
	expMethod := "list"
	if entries[0]["grpc.method"] != expMethod {
		t.Errorf("unexpected grpc method; expected %s, got %s", expMethod, entries[0]["grpc.method"])
	}
	expMsg := "handled gRPC unary request"
	if entries[0]["message"] != expMsg {
		t.Errorf("unexpected log message; expected %s, got %s", expMsg, entries[0]["message"])
	}
	if entries[0]["grpc.duration"] == nil {
		t.Errorf("missing grpc.duration field")
	}
	if entries[0]["requestid"] == nil {
		t.Errorf("missing requestid field")
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
	logWriter := bytes.Buffer{}
	interceptor := g.LogStreamInterceptor(
		zerolog.New(&logWriter),
		"requestid",
	)

	err := interceptor(
		"request",
		&g.WrappedServerStream{WrappedContext: metadata.NewIncomingContext(context.Background(), metadata.MD{})},
		&grpc.StreamServerInfo{FullMethod: "list"},
		handler,
	)
	if err != nil {
		t.Fatal(err)
	}
	entries := logEntriesFromBuffer(logWriter)
	expMethod := "list"
	if entries[0]["grpc.method"] != expMethod {
		t.Errorf("unexpected grpc method; expected %s, got %s", expMethod, entries[0]["grpc.method"])
	}
	expMsg := "handled gRPC stream request"
	if entries[0]["message"] != expMsg {
		t.Errorf("unexpected log message; expected %s, got %s", expMsg, entries[0]["message"])
	}
	if entries[0]["grpc.duration"] == nil {
		t.Errorf("missing grpc.duration field")
	}
	if entries[0]["requestid"] == nil {
		t.Errorf("missing requestid field")
	}
}

func logEntriesFromBuffer(buff bytes.Buffer) []map[string]interface{} {
	parts := strings.Split(buff.String(), "\n")
	var entries []map[string]interface{}
	for i, e := range parts {
		entries = append(entries, map[string]interface{}{})
		err := json.Unmarshal([]byte(e), &entries[i])
		if err != nil {
			log.Print(err)
		}
	}
	return entries
}
