package tracing

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracerHandler interface for OpenTelemetry tracing.
type TracerHandler interface {
	Start(resource *resource.Resource)
	Stop()
}

// NewHandler returns a new tracer handler instance.
func NewHandler() TracerHandler {
	return &tracerHandlerImpl{}
}

type tracerHandlerImpl struct {
	provider *sdktrace.TracerProvider
}

func (t *tracerHandlerImpl) Start(resource *resource.Resource) {
	if t == nil || !features.Tracing.Enabled() {
		return
	}

	provider, err := newProvider(resource)
	if err != nil {
		utils.Should(err)
		return
	}
	t.provider = provider
}

func (t *tracerHandlerImpl) Stop() {
	if t == nil || !features.Tracing.Enabled() {
		return
	}

	if err := t.provider.Shutdown(context.Background()); err != nil {
		utils.Should(err)
	}
}

func newTraceExporter() (sdktrace.SpanExporter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		env.OpenTelemetryCollectorURL.Setting(),
		// TODO: Add TLS.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gRPC connection to collector")
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gRPC exporter")
	}
	return traceExporter, nil
}

func newProvider(resource *resource.Resource) (*sdktrace.TracerProvider, error) {
	traceExporter, err := newTraceExporter()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create trace exporter")
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(resource),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.Baggage{},
			propagation.TraceContext{},
		),
	)
	return tp, nil
}
