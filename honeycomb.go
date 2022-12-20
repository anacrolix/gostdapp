package app

import (
	"context"
	"fmt"
	"os"

	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Automatic Honeycomb instrumentation, per
// https://docs.honeycomb.io/getting-data-in/opentelemetry/go-distro/#automatic-instrumentation.
func ConfigureOpenTelemetry() (cleanup func(), err error) {
	cleanup = func() {}
	resourceAttrs := make(map[string]string)
	iterFlyAttrs(func(key, value string) {
		resourceAttrs[key] = value
	})
	// Makes use of https://github.com/honeycombio/honeycomb-opentelemetry-go.
	otelShutdown, err := launcher.ConfigureOpenTelemetry(
		launcher.WithResourceAttributes(resourceAttrs),
		//// Try to shake out marshalling errors. Doesn't seem to help.
		//launcher.WithExporterProtocol(launcher.ProtocolHTTPProto),
	)
	if err != nil {
		err = fmt.Errorf("setting up OTel SDK: %w", err)
	} else {
		cleanup = otelShutdown
	}
	return
}

func iterFlyAttrs(f func(key, value string)) {
	emit := func(attrKey, envKey string) {
		if value, ok := os.LookupEnv(envKey); ok {
			f(attrKey, value)
		}
	}
	emit("fly.region", "FLY_REGION")
	emit("fly.alloc_id", "FLY_ALLOC_ID")
	emit("fly.app_name", "FLY_APP_NAME")
}

// Performs steps at
// https://docs.honeycomb.io/getting-data-in/opentelemetry/go-distro/#using-opentelemetry-without-the-honeycomb-distribution,
// doesn't automatically configure for Honeycomb.
func ConfigureOpenTelemetryManually(ctx context.Context) (cleanup func(), err error) {
	// Configure a new OTLP exporter using environment variables for sending data to Honeycomb over gRPC

	// https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlptrace
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		err = fmt.Errorf("creating exporter: %w", err)
		return
	}

	var flyAttrs []attribute.KeyValue
	iterFlyAttrs(func(key, value string) {
		flyAttrs = append(flyAttrs, attribute.String(key, value))
	})
	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewSchemaless(flyAttrs...)),
	)

	// Handle shutdown to ensure all sub processes are closed correctly and telemetry is exported
	cleanup = func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	// Register the W3C trace context and baggage propagators so data is propagated across services/processes
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return
}

// Performs steps at
// https://docs.honeycomb.io/getting-data-in/opentelemetry/go-distro/#using-opentelemetry-without-the-honeycomb-distribution,
// and applies Honeycomb-style configuration from environment.
func ConfigureOpenTelemetryForHoneycomb(ctx context.Context) (cleanup func(), err error) {
	// Configure a new OTLP exporter using environment variables for sending data to Honeycomb over gRPC

	// https://github.com/open-telemetry/opentelemetry-go/tree/main/exporters/otlp/otlptrace
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team": os.Getenv("HONEYCOMB_API_KEY"),
		}),
	)
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		err = fmt.Errorf("creating exporter: %w", err)
		return
	}

	var flyAttrs []attribute.KeyValue
	iterFlyAttrs(func(key, value string) {
		flyAttrs = append(flyAttrs, attribute.String(key, value))
	})
	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewSchemaless(flyAttrs...)),
	)

	// Handle shutdown to ensure all sub processes are closed correctly and telemetry is exported
	cleanup = func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	// Register the W3C trace context and baggage propagators so data is propagated across services/processes
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return
}
