package otel

import (
	"context"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SpanRecordError decorates a span with attributes and records the error
// SetStatus doesn't seem to work with Cloud Trace at the moment so we're using the '/http/status_code' attribute as that seems to be the only to colour a span red
// It also adds the error flag to make filtering traces with errors easier and a description of the error
func SpanRecordError(span trace.Span, err error, description string, eventOptions ...trace.EventOption) {
	span.SetAttributes(
		// This colours the span red in Cloud Trace
		attribute.Int("/http/status_code", 500),
		// This helps filter traces with error spans, we can use 'HasLabel:error' in Cloud Trace
		attribute.Bool("error", true),
	)
	eventOptions = append(
		eventOptions,
		trace.WithAttributes(attribute.String("exception.description", description)),
	)
	span.RecordError(err, eventOptions...)
}

var defaultGCPTraceLog = gcpTraceLog{
	spanFieldName:  "logging.googleapis.com/spanId",
	traceFieldName: "logging.googleapis.com/trace",
}

type gcpTraceLog struct {
	traceFieldName string
	spanFieldName  string
}

// LogFromCtx returns a log from the provided context. It adds GCP trace and span fields so the log can be associated with cloud tracing
func (tl *gcpTraceLog) LogFromCtx(ctx context.Context) *zerolog.Logger {
	log := zerolog.Ctx(ctx)
	span := trace.SpanFromContext(ctx)
	fields := map[string]interface{}{}
	if span.SpanContext().HasSpanID() {
		fields[tl.spanFieldName] = span.SpanContext().SpanID()
	}
	if span.SpanContext().HasTraceID() {
		fields[tl.traceFieldName] = span.SpanContext().TraceID()
	}
	l := log.With().Fields(fields).Logger()
	return &l
}

type noopTraceLog struct{}

// LogFromCtx
func (tl *noopTraceLog) LogFromCtx(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}
