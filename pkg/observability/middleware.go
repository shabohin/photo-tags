package observability

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates HTTP middleware for OpenTelemetry tracing
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from headers
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start span
			ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path,
				trace.WithAttributes(
					semconv.HTTPMethod(r.Method),
					semconv.HTTPRoute(r.URL.Path),
					semconv.HTTPScheme(r.URL.Scheme),
					semconv.ServerAddress(r.Host),
					semconv.UserAgentOriginal(r.UserAgent()),
				),
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			// Wrap response writer to capture status code
			wrapper := &responseWriter{ResponseWriter: w, statusCode: 200}

			// Process request
			start := time.Now()
			next.ServeHTTP(wrapper, r.WithContext(ctx))
			duration := time.Since(start)

			// Add response attributes
			span.SetAttributes(
				semconv.HTTPStatusCode(wrapper.statusCode),
				attribute.Int64("http.response_size", wrapper.bytesWritten),
				attribute.Float64("http.duration_ms", float64(duration.Nanoseconds())/1e6),
			)

			// Set span status based on HTTP status code
			if wrapper.statusCode >= 400 {
				span.SetAttributes(attribute.Bool("error", true))
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// TraceRabbitMQPublish traces RabbitMQ message publishing
func TraceRabbitMQPublish(
	ctx context.Context,
	serviceName, queueName string,
	message interface{},
) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)

	ctx, span := tracer.Start(ctx, "rabbitmq.publish",
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", queueName),
			attribute.String("messaging.operation", "publish"),
		),
		trace.WithSpanKind(trace.SpanKindProducer),
	)

	return ctx, span
}

// TraceRabbitMQConsume traces RabbitMQ message consumption
func TraceRabbitMQConsume(ctx context.Context, serviceName, queueName string) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)

	ctx, span := tracer.Start(ctx, "rabbitmq.consume",
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.source", queueName),
			attribute.String("messaging.operation", "receive"),
		),
		trace.WithSpanKind(trace.SpanKindConsumer),
	)

	return ctx, span
}
