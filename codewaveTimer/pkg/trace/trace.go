package tracing

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const traceIDKey = "trace-id"

func InitTracerProvider(url string) (*sdktrace.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("codewave-timer"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func NewSpan(ctx context.Context, tracerName string, spanName string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, spanName)
	return ctx, span
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func TraceIDFromContext(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey).(string)
	return traceID, ok
}
