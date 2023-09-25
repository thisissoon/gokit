# Kit

[![CircleCI](https://circleci.com/gh/thisissoon/gokit.svg?style=svg)](https://circleci.com/gh/thisissoon/gokit)
[![Go Report Card](https://goreportcard.com/badge/go.soon.build/kit)](https://goreportcard.com/report/go.soon.build/kit)

A set of common packages for building applications in Go at SOON_.
The packages are split into modules to enable granular control
over dependencies.

## Modules

### Config
Common configuration management with [viper](https://github.com/spf13/viper). Supports toml files, auto ENV var bindings and cobra command flag overrides.
```
go get go.soon.build/kit/config
```

### gRPC
Common helper constructs for running a gRPC server.
```
go get go.soon.build/kit/grpc
```

### HTTP
Common helper constructs for running a HTTP server, using the `http.Handler` pattern from the standard library.
```
go get go.soon.build/kit/http
```

### PSQL
Common helpers for managing postgres database connections and migrations.
```
go get go.soon.build/kit/psql
```

### PubSub
A super minimal publish/subscribe interface with backend implementations for different providers:
 - Google Cloud PubSub
```
go get go.soon.build/kit/pubsub
```

### Tracing

Small, opinionated wrappers and helpers for easily setting up tracing in a streamlined fashion.

Currently only supports Google Cloud's tracing service as a target, using OpenTelemtry.

```
go get go.soon.build/kit/tracing/otel
```

Please note that currently this package mainly revolves around setting up the OpenTelemtry SDK, and actual instrumentation of an application should (mostly) make use of existing packages such as otelhttp, otelgrpc, otel itself, etc.

The main helper method for actually creating spans is `OtelProvider.StartSpan` as there is no direct equivalent for creating a span directly from just a context object within the native SDK.

Here are some example snippets:

```go
// Initial setup
provider, err := NewOtelProvider("service-name",
    WithServiceNamespace("important-project"),
    WithServiceVersion("0.1.0"),
    WithGcpExporter("my-gcp-project-id"),
)
if err != nil {
    return err
}

cleanup, err := provider.SetupGlobalState(context.TODO())
if err != nil {
    return err
}
defer cleanup()

// Getting the current span from a context (normal OTEL SDK)
span := trace.SpanFromContext(ctx)
span.RecordError(...)
yada yada

// Creating a new span using a context (using this library)
ctx, span := provider.StartSpan(ctx, "span name")
defer span.End()

// Instrumenting a HTTP handler after setup is complete (otelhttp library)
handler := http.HandlerFunc(...)
wrapper := otelhttp.NewHandler(handler, "/myEndpoint")

// Instrumenting a HTTP request (otelhttp library)
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
    Timeout:   time.Second * 30,
}
req := http.NewRequestWithContext(ctx, ...)
res, err := client.Do(req)

// Instrumenting a GRPC server (otelgrpc library)
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        otelgrpc.UnaryServerInterceptor(),
    ),
    grpc.ChainStreamInterceptor(
        otelgrpc.StreamServerInterceptor(),
    ),
)

// Instrumenting a GRPC client (otelgrpc library)
conn, err := grpc.DialContext( // Alternatively: grpckit.NewClient
    ctx,
    addr,
    grpc.WithChainStreamInterceptor(otelgrpc.StreamClientInterceptor()),
    grpc.WithChainUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
)

// Creating a dummy tracer for unittests (normal OTEL SDK)
tp := trace.NewNoopTracerProvider().Tracer("")

// Recording an error in a way that GCP Cloud Trace likes (this library)
err := errors.New("blah")
span := ...
otelkit.SpanRecordError(span, err, "when decoding JSON")

// Preventing traces from occurring on certain grpc endpoints (this library)
srv := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        otelgrpc.UnaryServerInterceptor(
            otelgrpc.WithInterceptorFilter(
                otelkit.FilterMethods("/abc.Service/Method", "/def.Service/Method")
            )
        ),
    ),
)

// linking logs to a span
provider, err := NewOtelProvider("service-name",
	otelkit.WithGCPTraceLogger(),
)
ctx, span := s.tracer.Start(stream.Context(), "UpdateRedirects")
defer span.End()
log := zerolog.Ctx(ctx)
log.Info().Msg("this will including the span and trace id")
```

## Development

### Tests

To run the test suite with coverage report:
```
make test
```

To run pubsub tests with gcloud emulator:
```bash
❯ gcloud beta emulators pubsub start
...
❯ $(gcloud beta emulators pubsub env-init)
❯ make testgcloud
```
