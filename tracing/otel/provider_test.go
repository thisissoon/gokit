package otel

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var testExporter *tracetest.InMemoryExporter
var flushTestSpans func()
var testProvider *OtelProvider
var providerMutex sync.Mutex

func TestMain(m *testing.M) {
	var err error
	testExporter = tracetest.NewInMemoryExporter()
	testProvider, err = NewOtelProvider("test")
	if err != nil {
		panic(err)
	}
	testProvider.exporter = testExporter

	flushTestSpans, err = testProvider.SetupGlobalState(context.Background())
	if err != nil {
		panic(err)
	}
	defer flushTestSpans()

	code := m.Run()
	os.Exit(code)
}

func TestOptions(t *testing.T) {
	serviceName := "test-service"
	cases := map[string]struct {
		options    []OtelProviderOption
		expected   *OtelProvider
		customTest func(*testing.T, *OtelProvider, error)
	}{
		"WithServiceNamespace": {
			options: []OtelProviderOption{WithServiceNamespace("test-namespace")},
			expected: &OtelProvider{
				serviceName:      serviceName,
				serviceNamespace: "test-namespace",
				getTraceLogger:   &noopTraceLog{},
			},
		},
		"WithServiceVersion": {
			options: []OtelProviderOption{WithServiceVersion("v1.0.0")},
			expected: &OtelProvider{
				serviceName:    serviceName,
				serviceVersion: "v1.0.0",
				getTraceLogger: &noopTraceLog{},
			},
		},
		"WithTracerName": {
			options: []OtelProviderOption{WithTracerName("go.soon.build/kit/tracing/otel")},
			expected: &OtelProvider{
				serviceName:    serviceName,
				tracerName:     "go.soon.build/kit/tracing/otel",
				getTraceLogger: &noopTraceLog{},
			},
		},
		"WithGlobalAttributes": {
			options: []OtelProviderOption{
				WithGlobalAttributes(
					attribute.Bool("some-bool", true),
				),
				WithGlobalAttributes(
					attribute.Bool("some-other-bool", false),
				),
			},
			expected: &OtelProvider{
				serviceName: serviceName,
				globalAttributes: []attribute.KeyValue{
					{
						Key:   "some-bool",
						Value: attribute.BoolValue(true),
					},
					{
						Key:   "some-other-bool",
						Value: attribute.BoolValue(false),
					},
				},
				getTraceLogger: &noopTraceLog{},
			},
		},
		"WithResourceOptions": {
			options: []OtelProviderOption{
				WithResourceOptions(
					resource.WithOS(),
				),
				WithResourceOptions(
					resource.WithOSDescription(),
				),
			},
			expected: &OtelProvider{
				serviceName: serviceName,
				resourceOptions: []resource.Option{
					resource.WithOS(),
					resource.WithOSDescription(),
				},
				getTraceLogger: &noopTraceLog{},
			},
		},
		"WithTracerProviderOptions": {
			// TracerProviderOptions appear to never be equal with `assert.Equal`
			options: []OtelProviderOption{
				WithTracerProviderOptions(
					trace.WithRawSpanLimits(trace.NewSpanLimits()),
				),
				WithTracerProviderOptions(
					trace.WithRawSpanLimits(trace.NewSpanLimits()),
				),
			},
			customTest: func(t *testing.T, op *OtelProvider, err error) {
				assert.Nil(t, err)
				assert.Len(t, op.tracerProviderOptions, 2)
			},
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			provider, err := NewOtelProvider(serviceName, testCase.options...)
			if testCase.customTest != nil {
				testCase.customTest(t, provider, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, testCase.expected, provider)
			}
		})
	}
}

func TestOptionalResourceOptionsAreCreated(t *testing.T) {
	serviceName := "test-service"
	cases := map[string]struct {
		option         OtelProviderOption
		expectedAttrib attribute.KeyValue
	}{
		"Service Name (created by default)": {
			option: func(op *OtelProvider) error { return nil },
			expectedAttrib: attribute.KeyValue{
				Key:   semconv.ServiceNameKey,
				Value: attribute.StringValue(serviceName),
			},
		},
		"WithServiceNamespace": {
			option: WithServiceNamespace("namespace"),
			expectedAttrib: attribute.KeyValue{
				Key:   semconv.ServiceNamespaceKey,
				Value: attribute.StringValue("namespace"),
			},
		},
		"WithServiceVersion": {
			option: WithServiceVersion("version"),
			expectedAttrib: attribute.KeyValue{
				Key:   semconv.ServiceVersionKey,
				Value: attribute.StringValue("version"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			provider, err := NewOtelProvider(serviceName, tc.option)
			assert.Nil(t, err)
			assert.NotNil(t, provider)

			res, err := provider.createResource(context.TODO())
			assert.Nil(t, err)
			assert.Contains(t, res.Attributes(), tc.expectedAttrib)
		})
	}
}

func TestGetTracerName(t *testing.T) {
	provider, err := NewOtelProvider("service", WithTracerName("tracer"))
	assert.Nil(t, err)
	assert.NotNil(t, provider)

	assert.Equal(t, "tracer", provider.getTracerName())
	provider.tracerName = ""
	assert.Equal(t, "service", provider.getTracerName())
}

func TestStart(t *testing.T) {
	providerMutex.Lock()
	defer providerMutex.Unlock()
	testExporter.Reset()
	mw := bytes.NewBufferString("")
	testLog := zerolog.New(mw).With().Bool("testLogger", true).Logger()
	testProvider.getTraceLogger = &mockTraceLogger{
		ret: &testLog,
	}
	ctx, span := testProvider.Start(context.Background(), "testSpan")
	assert.NotNil(t, ctx)
	assert.True(t, span.IsRecording())
	span.End()
	flushTestSpans()

	assert.Len(t, testExporter.GetSpans(), 1)

	l := zerolog.Ctx(ctx)
	l.Info().Send()
	assert.Equal(t, `{"level":"info","testLogger":true}
`, mw.String())
}

func TestSamplerFromEnv(t *testing.T) {
	providerMutex.Lock()
	defer func() {
		os.Unsetenv(otelSamplerEnvVar)
		os.Unsetenv(otelSamplerArgEnvVar)
		providerMutex.Unlock()
	}()

	cases := map[string]struct {
		sampler      string
		samplerArg   string
		descContains string
		shouldError  bool
	}{
		"traceidratio": {
			sampler:      "traceidratio",
			descContains: "AlwaysOnSampler", // quirk: traceidratio becomes `AlwaysOnSampler` when arg is 1.0
		},
		"traceidratio_sampled": {
			sampler:      "traceidratio",
			samplerArg:   "0.99",
			descContains: "TraceIDRatioBased",
		},
		"always_off": {
			sampler:      "always_off",
			descContains: "AlwaysOffSampler",
		},
		"always_on": {
			sampler:      "always_on",
			descContains: "AlwaysOnSampler",
		},
		"default sampler": {
			descContains: "AlwaysOnSampler",
		},
		"parentbased_traceidratio": {
			sampler:      "parentbased_traceidratio",
			descContains: "AlwaysOnSampler", // quirk: traceidratio becomes `AlwaysOnSampler` when arg is 1.0
		},
		"parentbased_traceidratio_sampled": {
			sampler:      "parentbased_traceidratio",
			samplerArg:   "0.99",
			descContains: "TraceIDRatioBased",
		},
		"parentbased_always_off": {
			sampler:      "parentbased_always_off",
			descContains: "AlwaysOffSampler",
		},
		"parentbased_always_on": {
			sampler:      "parentbased_always_on",
			descContains: "AlwaysOnSampler",
		},
		"float parse error": {
			sampler:     "traceidratio",
			samplerArg:  "l.01",
			shouldError: true,
		},
		"unknown sampler error": {
			sampler:     "foo",
			shouldError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			os.Setenv(otelSamplerEnvVar, tc.sampler)
			os.Setenv(otelSamplerArgEnvVar, tc.samplerArg)
			sampler, err := samplerFromEnv()

			if tc.shouldError {
				assert.Error(t, err)
				assert.Nil(t, sampler)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sampler)
				assert.Contains(t, sampler.Description(), tc.descContains)
			}
		})
	}
}

func TestTracerFromContext(t *testing.T) {
	cases := map[string]struct {
		IsRemote         bool
		ExpectNoopTracer bool
	}{
		"remote spans should use the global trace provider": {
			IsRemote:         true,
			ExpectNoopTracer: false,
		},
		"non-remote spans should use the span's trace provider": {
			IsRemote:         false,
			ExpectNoopTracer: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			spanContext := spanContext(t, testTraceID, testSpanID, tc.IsRemote)
			assert.True(t, spanContext.IsValid())
			assert.Equal(t, tc.IsRemote, spanContext.IsRemote())

			ctx := oteltrace.ContextWithSpan(context.Background(), &mockSpan{
				spanContext: spanContext,
			})
			assert.Equal(t, spanContext, oteltrace.SpanContextFromContext(ctx))

			// Ensure that tracerFromContext decides to use the global provider.
			// The mock SpanContext produces `noop.Tracer` while the global provider produces `trace.tracer`
			_, isNoopTracer := testProvider.tracerFromContext(ctx).(noop.Tracer)
			assert.Equal(t, tc.ExpectNoopTracer, isNoopTracer)
		})
	}
}

func TestPrometheusExporter(t *testing.T) {
	// Allow otelkit to setup a Prometheus exporter, avoiding mutation of global state.
	opt, handler := WithPrometheusMetricExporter()
	otel, err := NewOtelProvider("prometheus", opt, WithTracerName("test"))
	assert.NoError(t, err)
	otel.createMeterProvider(&resource.Resource{})

	// Setup test server & test meter
	server := httptest.NewServer(handler)
	client := server.Client()
	defer server.Close()

	meter := otel.Meter()
	counter, err := meter.Int64Counter("counter")
	assert.NoError(t, err)
	counter.Add(context.Background(), 1)

	// Ensure the metric is visible
	resp, err := client.Get(server.URL)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "# TYPE counter_total counter") // _total is added automatically onto counters.
}

type mockTraceLogger struct {
	ret *zerolog.Logger
}

func (m *mockTraceLogger) LogFromCtx(ctx context.Context) *zerolog.Logger {
	return m.ret
}
