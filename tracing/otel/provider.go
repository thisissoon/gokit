package otel

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	gcptraceexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"google.golang.org/api/option"
)

// Provides an opinionated wrapper around the Open Telemtry SDK.
type OtelProvider struct {
	embedded.Tracer

	serviceName      string
	serviceNamespace string
	serviceVersion   string
	tracerName       string

	globalAttributes      []attribute.KeyValue
	resourceOptions       []resource.Option
	tracerProviderOptions []sdktrace.TracerProviderOption
	metricProviderOptions []sdkmetric.Option

	exporter      sdktrace.SpanExporter
	metricReader  sdkmetric.Reader
	meterProvider *sdkmetric.MeterProvider

	getTraceLogger  getTraceLogger
	afterSetupFuncs []afterSetupFunc
}

// Contains information for setting up a HTTP server, which some metrics exporters (prometheus)
// may need to do.
type MetricServerInfo struct {
	Host string // The host to bind the server to, e.g. ":3000"
	Path string // The path to serve the metrics endpoint on, e.g. "/metrics"
}

var _ trace.Tracer = &OtelProvider{}

// Typical option function pattern
type OtelProviderOption func(*OtelProvider) error

// Cleanup function to be defer-called when returned
type CleanupFunc func()

const (
	otelSamplerEnvVar    = "OTEL_TRACES_SAMPLER"
	otelSamplerArgEnvVar = "OTEL_TRACES_SAMPLER_ARG"
)

// Constructs a new OtelProvider using the given options to configure the instance.
//
// If an option function returns an error, an error is returned alongside a null provider.
func NewOtelProvider(serviceName string, opts ...OtelProviderOption) (*OtelProvider, error) {
	provider := new(OtelProvider)
	provider.serviceName = serviceName
	provider.getTraceLogger = &noopTraceLog{}

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

// Sets the trace logger to use GCP
func WithGCPTraceLogger() OtelProviderOption {
	return func(op *OtelProvider) error {
		tl := &defaultGCPTraceLog
		op.getTraceLogger = tl
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
func WithGcpExporterAndOptions(clientOpts []option.ClientOption, opts ...gcptraceexporter.Option) OtelProviderOption {
	return func(op *OtelProvider) error {
		// Build clientOptions with initial options and disable telemetry
		clientOptions := append([]option.ClientOption{option.WithTelemetryDisabled()}, clientOpts...)

		// build tracing client
		tracingClient := gcptraceexporter.WithTraceClientOptions(clientOptions)

		// add tracing client to exporter options
		opts = append(opts, tracingClient)

		// build exporter
		exporter, err := gcptraceexporter.New(opts...)
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
	return WithGcpExporterAndOptions(nil, gcptraceexporter.WithProjectID(projectId))
}

// Metrics are created via an integration with the Prometheus SDK. Metrics can be
// "exported" by serving the returned handler on a HTTP server.
//
// You may prefer to use `WithPrometheusMetricExporterAutoServer` for ease of use.
func WithPrometheusMetricExporter(opt ...prometheus.Option) (OtelProviderOption, http.Handler) {
	return func(op *OtelProvider) error {
		exporter, err := prometheus.New(opt...)
		if err != nil {
			return err
		}
		op.metricReader = exporter

		return nil
	}, promhttp.Handler()
}

// Similar to `WithPrometheusMetricExporter` except this will automatically setup a HTTP server
// after `SetupGlobalState` is called.
func WithPrometheusMetricExporterAutoServer(server MetricServerInfo, opt ...prometheus.Option) OtelProviderOption {
	return func(op *OtelProvider) error {
		optFunc, handler := WithPrometheusMetricExporter(opt...)
		if err := optFunc(op); err != nil {
			return err
		}

		op.afterSetupFuncs = append(op.afterSetupFuncs, func() error {
			go func() {
				mux := http.NewServeMux()
				mux.Handle(server.Path, handler)
				if err := http.ListenAndServe(server.Host, mux); err != nil {
					fmt.Printf("error starting prometheus server: %v", err) // TODO: Use a proper logger?
				}
			}()
			return nil
		})

		return nil
	}
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
// A new MeterProvider is created using certain defaults.
//
// The MeterProvider uses the aforementioned Resource as its default.
//
// The MeterProvider is registered as the global provider within the OTEL SDK.
//
// A TextMapPropagator for the W3C Trace Context and B3 formats is installed by default as the
// global propagator.
//
// Some exporters may perform additional actions during this function, for example `WithPrometheusMetricExporterAutoServer`
// will start its HTTP server.
//
// You should defer-call the returned `CleanupFunc` as this will force the span batcher to flush
// the spans/metrics into the underlying exporter.
func (o *OtelProvider) SetupGlobalState(ctx context.Context) (CleanupFunc, error) {
	if o.exporter == nil {
		o.exporter = tracetest.NewNoopExporter() // To eliminate any potential nil access errors
	}
	if o.metricReader == nil {
		o.metricReader = sdkmetric.NewManualReader()
	}

	res, err := o.createResource(ctx)
	if err != nil {
		return func() {}, err
	}

	sampler, err := samplerFromEnv()
	if err != nil {
		return func() {}, err
	}

	cleanupFuncs := []func(){}

	opts := append(
		o.tracerProviderOptions,
		sdktrace.WithBatcher(o.exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	provider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(provider)
	cleanupFuncs = append(cleanupFuncs, func() { _ = provider.ForceFlush(ctx) })

	o.createMeterProvider(res)
	otel.SetMeterProvider(o.meterProvider)
	cleanupFuncs = append(cleanupFuncs, func() { _ = o.meterProvider.ForceFlush(ctx) })

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
		),
	)

	for _, fun := range o.afterSetupFuncs {
		if err = fun(); err != nil {
			return nil, err
		}
	}

	return func() {
		for _, fun := range cleanupFuncs {
			fun()
		}
	}, nil
}

// A simple helper function that retrieves the current tracer from the context (or fetches a global Tracer)
// and then uses it to start a new span.
//
// If you're interested in further details, please see the `Tracer.Start` function from the OTEL SDK.
func (o *OtelProvider) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx, span := o.tracerFromContext(ctx).Start(ctx, spanName, opts...)
	log := o.getTraceLogger.LogFromCtx(ctx)
	ctx = log.WithContext(ctx)
	return ctx, span
}

// A loose wrapper around the standard OTEL MeterProvider.Meter function using the previously
// configured MeterProvider, which also sets a few default options.
//
// If you're interested in further details, please see the `MeterProvider.Meter` function from the OTEL SDK.
func (o *OtelProvider) Meter(opts ...metric.MeterOption) metric.Meter {
	return o.meterProvider.Meter(o.getTracerName(), opts...)
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
		// resource.WithTelemetrySDK(),
		resource.WithAttributes(attribs...),
	)

	return resource.New(ctx, opts...)
}

func (o *OtelProvider) createMeterProvider(res *resource.Resource) {
	mpOpts := append(
		o.metricProviderOptions,
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(o.metricReader),
	)
	o.meterProvider = sdkmetric.NewMeterProvider(mpOpts...)
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
//
// Remote spans will cause this function to always create a new tracer, as it
// is very likely that the TracerProvider attached to the remote span is a noop one,
// causing incorrect behaviour.
func (o *OtelProvider) tracerFromContext(ctx context.Context) trace.Tracer {
	var provider trace.TracerProvider

	if span := trace.SpanFromContext(ctx); span != nil && span.SpanContext().IsValid() && !span.SpanContext().IsRemote() {
		provider = span.TracerProvider()
	}
	if provider == nil {
		provider = otel.GetTracerProvider()
	}

	return provider.Tracer(o.getTracerName())
}

type getTraceLogger interface {
	LogFromCtx(ctx context.Context) *zerolog.Logger
}

type afterSetupFunc func() error

// Despite OTEL SDK's documentation, it doesn't seem to want to use the standard
// OTEL env vars when deployed into GKE. Instead of trying to debug/find
// the obscure piece of documentation on why this happens, we've instead
// decided to handle some of the env vars ourself.
func samplerFromEnv() (sdktrace.Sampler, error) {
	sampler := os.Getenv(otelSamplerEnvVar)
	samplerArg := os.Getenv(otelSamplerArgEnvVar)
	samplerArgFloat := 1.0

	if samplerArg != "" {
		var err error
		samplerArgFloat, err = strconv.ParseFloat(samplerArg, 32)
		if err != nil {
			return nil, err
		}
	}

	switch sampler {
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(samplerArgFloat), nil
	case "always_off":
		return sdktrace.NeverSample(), nil
	case "always_on":
		return sdktrace.AlwaysSample(), nil
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplerArgFloat)), nil
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	case "":
		return sdktrace.AlwaysSample(), nil
	default:
		return nil, fmt.Errorf("unknown value for %s: %s", otelSamplerEnvVar, sampler)
	}
}
