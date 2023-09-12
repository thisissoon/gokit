package otel

import (
	"context"

	gcpexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Provides an opinionated wrapper around the Open Telemtry SDK.
type OtelProvider struct {
	serviceName      string
	serviceNamespace string
	serviceVersion   string
	tracerName       string

	globalAttributes      []attribute.KeyValue
	resourceOptions       []resource.Option
	tracerProviderOptions []sdktrace.TracerProviderOption

	exporter sdktrace.SpanExporter
}

var _ trace.Tracer = &OtelProvider{}

// Typical option function pattern
type OtelProviderOption func(*OtelProvider) error

// Cleanup function to be defer-called when returned
type CleanupFunc func()

// Constructs a new OtelProvider using the given options to configure the instance.
//
// If an option function returns an error, an error is returned alongside a null provider.
func NewOtelProvider(serviceName string, opts ...OtelProviderOption) (*OtelProvider, error) {
	provider := new(OtelProvider)
	provider.serviceName = serviceName

	for _, opt := range opts {
		if err := opt(provider); err != nil {
			return nil, err
		}
	}

	return provider, nil
}

// Specifies the namespace of the service. We usually use the name of the overarching
// project here.
func WithServiceNamespace(namespace string) OtelProviderOption {
	return func(op *OtelProvider) error {
		op.serviceNamespace = namespace
		return nil
	}
}

// Specifies the version of the service. Useful if there's more than one version of the
// service running at any given time.
func WithServiceVersion(version string) OtelProviderOption {
	return func(op *OtelProvider) error {
		op.serviceVersion = version
		return nil
	}
}

// Sets the name of the underlying tracer that is used to create spans.
//
// This isn't the most important thing in the world, and if not specified then
// the service name is used as a fallback.
func WithTracerName(name string) OtelProviderOption {
	return func(op *OtelProvider) error {
		op.tracerName = name
		return nil
	}
}

// Appends the global set of attributes that are attached onto every span.
func WithGlobalAttributes(attribs ...attribute.KeyValue) OtelProviderOption {
	return func(op *OtelProvider) error {
		op.globalAttributes = append(op.globalAttributes, attribs...)
		return nil
	}
}

// Appends any additional options to use when creating the default OTEL resource.
// Note that the options set by this library take priority.
func WithResourceOptions(opts ...resource.Option) OtelProviderOption {
	return func(op *OtelProvider) error {
		op.resourceOptions = append(op.resourceOptions, opts...)
		return nil
	}
}

// Appends any additional options to use when creating the underlying trace provider.
// Note that the options set by this library take priority.
func WithTracerProviderOptions(opts ...sdktrace.TracerProviderOption) OtelProviderOption {
	return func(op *OtelProvider) error {
		op.tracerProviderOptions = append(op.tracerProviderOptions, opts...)
		return nil
	}
}

// Export spans into GCP's tracing service. This function is useful when you need
// to set any additional options when creating the exporter.
//
// Generally you can use `WithGcpExporter` instead as it has a simpler interface.
//
// A small side effect of this function is that it will add additional resource options
// (as if it called `WithResourceOptions`), which shouldn't really cause any issues.
func WithGcpExporterAndOptions(opts ...gcpexporter.Option) OtelProviderOption {
	return func(op *OtelProvider) error {
		exporter, err := gcpexporter.New(opts...)
		op.exporter = exporter
		op.resourceOptions = append(
			op.resourceOptions,
			resource.WithDetectors(gcp.NewDetector()),
		)
		return err
	}
}

// A simple wrapper around `WithGcpExporterAndOptions` for the common use case where
// only a GCP project ID needs to be provided.
func WithGcpExporter(projectId string) OtelProviderOption {
	return WithGcpExporterAndOptions(gcpexporter.WithProjectID(projectId))
}

// Sets up the global OTEL SDK state to use the specified configuration, with sane-ish defaults.
//
// A new Resource is created using certain defaults as well as anything passed in from `WithResourceOptions`.
//
// A new TracerProvider is created using certain defaults as well as anything passed in from `WithTracerProviderOptions`.
//
// The TracerProvider uses the aforementioned Resource as its default.
//
// The TracerProvider is registered as the global provider within the OTEL SDK.
//
// A TextMapPropagator for the W3C Trace Context and B3 formats is installed by default as the
// global propagator.
//
// You should defer-call the returned `CleanupFunc` as this will force the span batcher to flush
// the spans into the underlying exporter.
func (o *OtelProvider) SetupGlobalState(ctx context.Context) (CleanupFunc, error) {
	if o.exporter == nil {
		o.exporter = tracetest.NewNoopExporter() // To eliminate any potential nil access errors
	}

	res, err := o.createResource(ctx)
	if err != nil {
		return func() {}, err
	}

	opts := append(
		o.tracerProviderOptions,
		sdktrace.WithBatcher(o.exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)
	provider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
		),
	)

	return func() { provider.ForceFlush(ctx) }, nil
}

// A simple helper function that retrieves the current tracer from the context (or fetches a global Tracer)
// and then uses it to start a new span.
//
// If you're interested in further details, please see the `Tracer.Start` function from the OTEL SDK.
func (o *OtelProvider) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return o.tracerFromContext(ctx).Start(ctx, spanName, opts...)
}

// Creates a new resource using a bunch of the configuration options provided, as well
// as certain defaults.
func (o *OtelProvider) createResource(ctx context.Context) (*resource.Resource, error) {
	attribs := append(
		o.globalAttributes,
		semconv.ServiceName(o.serviceName),
	)

	if o.serviceNamespace != "" {
		attribs = append(attribs, semconv.ServiceNamespace(o.serviceNamespace))
	}
	if o.serviceVersion != "" {
		attribs = append(attribs, semconv.ServiceVersion(o.serviceVersion))
	}

	opts := append(
		o.resourceOptions,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(attribs...),
	)

	return resource.New(ctx, opts...)
}

// Returns either the configured tracer name, or the service name if no tracer name
// was explicitly set.
func (o *OtelProvider) getTracerName() string {
	if o.tracerName == "" {
		return o.serviceName
	}
	return o.tracerName
}

// Retrieves a tracer from the current span in the context, or creates/fetches
// a tracer based off of our configured tracer name.
func (o *OtelProvider) tracerFromContext(ctx context.Context) trace.Tracer {
	var provider trace.TracerProvider

	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		provider = span.TracerProvider()
	} else {
		provider = otel.GetTracerProvider()
	}

	return provider.Tracer(o.getTracerName())
}
