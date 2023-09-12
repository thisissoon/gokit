package otel

import (
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
