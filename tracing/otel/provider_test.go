package otel

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
			},
		},
		"WithServiceVersion": {
			options: []OtelProviderOption{WithServiceVersion("v1.0.0")},
			expected: &OtelProvider{
				serviceName:    serviceName,
				serviceVersion: "v1.0.0",
			},
		},
		"WithTracerName": {
			options: []OtelProviderOption{WithTracerName("go.soon.build/kit/tracing/otel")},
			expected: &OtelProvider{
				serviceName: serviceName,
				tracerName:  "go.soon.build/kit/tracing/otel",
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

	ctx, span := testProvider.Start(context.Background(), "testSpan")
	assert.NotNil(t, ctx)
	assert.True(t, span.IsRecording())
	span.End()
	flushTestSpans()

	assert.Len(t, testExporter.GetSpans(), 1)
}
