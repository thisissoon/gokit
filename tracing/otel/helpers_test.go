package otel

import (
	"bytes"
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

const (
	testTraceID = "0102030405060708090a0b0c0d0e0f10"
	testSpanID  = "0102030405060708"
)

func Test_gcpTraceLog_LogFromCtx(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := map[string]struct {
		gcpTraceLog *gcpTraceLog
		args        args
		traceID     string
		spanID      string
		xLogs       string
	}{
		"add spanId and traceId to the log": {
			gcpTraceLog: &gcpTraceLog{
				traceFieldName: "traceFieldName",
				spanFieldName:  "spanFieldName",
			},
			args: args{
				ctx: context.Background(),
			},
			traceID: testTraceID,
			spanID:  testSpanID,
			xLogs: `{"level":"info","spanFieldName":"0102030405060708","traceFieldName":"0102030405060708090a0b0c0d0e0f10"}
`,
		},
		"not add traceID if it doesn't exist": {
			gcpTraceLog: &gcpTraceLog{
				traceFieldName: "traceFieldName",
				spanFieldName:  "spanFieldName",
			},
			args: args{
				ctx: context.Background(),
			},
			traceID: "",
			spanID:  testSpanID,
			xLogs: `{"level":"info","spanFieldName":"0102030405060708"}
`,
		},
		"not add spanID if it doesn't exist": {
			gcpTraceLog: &gcpTraceLog{
				traceFieldName: "traceFieldName",
				spanFieldName:  "spanFieldName",
			},
			args: args{
				ctx: context.Background(),
			},
			traceID: testTraceID,
			spanID:  "",
			xLogs: `{"level":"info","traceFieldName":"0102030405060708090a0b0c0d0e0f10"}
`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockSpan := &mockSpan{
				spanContext: spanContext(t, tt.traceID, tt.spanID),
			}
			ctx := trace.ContextWithSpan(tt.args.ctx, mockSpan)
			mw := bytes.NewBufferString("")
			log := zerolog.New(mw)
			ctx = log.WithContext(ctx)
			got := tt.gcpTraceLog.LogFromCtx(ctx)
			got.Info().Msg("")
			assert.Equal(t, tt.xLogs, mw.String())
		})
	}
}

func spanContext(t *testing.T, traceID, spanID string) trace.SpanContext {
	spanCtx := trace.SpanContextConfig{}
	if traceID != "" {
		id, err := trace.TraceIDFromHex(traceID)
		if err != nil {
			t.Fatal(err)
		}
		spanCtx.TraceID = id
	}
	if spanID != "" {
		id, err := trace.SpanIDFromHex(spanID)
		if err != nil {
			t.Fatal(err)
		}
		spanCtx.SpanID = id
	}
	return trace.NewSpanContext(spanCtx)
}

type mockSpan struct {
	spanContext trace.SpanContext

	trace.Span
}

func (m *mockSpan) SpanContext() trace.SpanContext {
	return m.spanContext
}
