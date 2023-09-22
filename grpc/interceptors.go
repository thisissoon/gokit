package grpc

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TraceField provides the field names for logging the trace
type TraceField struct {
	// RequestFieldName is the name of the trace id field in the request e.g. x-b3-traceid
	RequestFieldName string
	// LoggingFieldName is the name of the trace field to send to the logger e.g. logging.googleapis.com/trace
	LoggingFieldName string
}

// RequestID extracts the request id from context, if there is
// no request id a fresh ID is generated
func RequestID(ctx context.Context, fieldName string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return xid.New().String()
	}
	if v, ok := md[fieldName]; ok && len(v) > 0 {
		return v[0]
	}
	return xid.New().String()
}

// TraceID extracts the trace ID from the context, if there is
// no trace id an empty string is returned
func TraceID(ctx context.Context, fieldName string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if v, ok := md[fieldName]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

// LogUnaryInterceptor returns grpc middleware to log unary method calls
func LogUnaryInterceptor(l zerolog.Logger, fieldName string, tf TraceField) grpc.UnaryServerInterceptor {
	tf = tf.mergeWithDefaults()
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var start = time.Now().UTC()
		log := l.With().Fields(map[string]interface{}{
			fieldName:           RequestID(ctx, fieldName),
			tf.LoggingFieldName: TraceID(ctx, tf.RequestFieldName),
			"grpc.method":       info.FullMethod,
		}).Logger()
		ctx = log.WithContext(ctx)
		defer log.Debug().
			TimeDiff("grpc.duration", time.Now().UTC(), start).
			Msg("handled gRPC unary request")
		return handler(ctx, req)
	}
}

// WrappedServerStream is a thin wrapper around grpc.ServerStream
// that allows modifying context
type WrappedServerStream struct {
	grpc.ServerStream

	WrappedContext context.Context
}

// Context returns the wrapper's WrappedContext,
// overwriting the nested grpc.ServerStream.Context()
func (w *WrappedServerStream) Context() context.Context {
	return w.WrappedContext
}

// LogStreamInterceptor returns grpc middleware to log stream method calls
func LogStreamInterceptor(l zerolog.Logger, fieldName string, tf TraceField) grpc.StreamServerInterceptor {
	tf = tf.mergeWithDefaults()
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var start = time.Now().UTC()
		log := l.With().Fields(map[string]interface{}{
			fieldName:           RequestID(ss.Context(), fieldName),
			tf.LoggingFieldName: TraceID(ss.Context(), tf.RequestFieldName),
			"grpc.method":       info.FullMethod,
		}).Logger()
		ctx := log.WithContext(ss.Context())
		ws := &WrappedServerStream{
			ss,
			ctx,
		}
		defer log.Debug().
			TimeDiff("grpc.duration", time.Now().UTC(), start).
			Msg("handled gRPC stream request")
		return handler(srv, ws)
	}
}

func (tf TraceField) mergeWithDefaults() TraceField {
	if tf.RequestFieldName == "" {
		tf.RequestFieldName = "x-b3-traceid"
	}
	if tf.LoggingFieldName == "" {
		tf.LoggingFieldName = "logging.googleapis.com/trace"
	}
	return tf
}
