package httpmiddleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/brokerclient/correlation"
	logger "gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetrics "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	CorrelationIdKey = "correlation-id"
	TelemetrySDKName = "go.opentelemetry.io/ote/sdk"
)

// NewObserveExporter creates an exporter to push metrics directly to Observe collector
func NewObserveExporter(ctx context.Context, collectorUrl string, token string) (*otlptrace.Exporter, error) {
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(collectorUrl),
		otlptracehttp.WithURLPath("/v1/otel/v1/traces"),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", token),
		}),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create observe exporter: %w", err)
	}

	return exporter, nil
}

// NewLocalMeterExporter creates an exporter to push metrics to a collector running in localhost
func NewLocalMeterExporter(ctx context.Context) (sdkmetrics.Exporter, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint("localhost:4317"),
		otlpmetricgrpc.WithCompressor("gzip"),
		otlpmetricgrpc.WithInsecure(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create local mettric exporter: %w", err)
	}

	return exporter, nil
}

// NewLocalTraceExporter creates an exporter to push traces to a collector running in localhost
func NewLocalTraceExporter(ctx context.Context) (*otlptrace.Exporter, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint("localhost:4317"),
		otlptracegrpc.WithCompressor("gzip"),
		otlptracegrpc.WithInsecure(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create local trace exporter: %w", err)
	}

	return exporter, nil
}

// InitOpenTelemetryTracer initializes an open telemetry tracer
func InitOpenTelemetryTracer(ctx context.Context,
	serviceName string,
	exporter *otlptrace.Exporter) func(context.Context) error {
	log := logger.FromContext(ctx)

	resources, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String(string(semconv.ServiceNameKey), serviceName),
			attribute.String(string(semconv.TelemetrySDKNameKey), TelemetrySDKName),
			attribute.String(string(semconv.TelemetrySDKLanguageKey), "go"),
		),
	)
	if err != nil {
		log.Panicf("Could not create open telemetry resource. Error: %v", err)

		return nil
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)
	return exporter.Shutdown
}

// InitOpenTelemetryMeter initializes an open telemetry meter
// interval is the interval which the metrics will be reported to the collector
func InitOpenTelemetryMeter(ctx context.Context,
	serviceName string,
	exporter sdkmetrics.Exporter,
	interval time.Duration,
) func(context.Context) error {
	log := logger.FromContext(ctx)

	resource, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String(string(semconv.ServiceNameKey), serviceName),
			attribute.String(string(semconv.TelemetrySDKNameKey), TelemetrySDKName),
			attribute.String(string(semconv.TelemetrySDKLanguageKey), "go"),
		),
	)
	if err != nil {
		log.Panicf("Could not create open telemetry resource. Error: %v", err)

		return nil
	}

	periodicReader := sdkmetrics.NewPeriodicReader(exporter,
		sdkmetrics.WithInterval(interval),
	)

	otel.SetMeterProvider(
		sdkmetrics.NewMeterProvider(
			sdkmetrics.WithReader(periodicReader),
			sdkmetrics.WithResource(resource),
		),
	)
	return exporter.Shutdown
}

// OpenTelemtryTraceMiddleware is a middleware that adds open telemetry traces for each http request.
func OpenTelemtryTraceMiddleware(chiRouter *chi.Mux) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			rCtx := chi.NewRouteContext()
			spanName := ""
			routePattern := ""
			if chiRouter.Match(rCtx, r.Method, r.URL.Path) {
				routePattern = rCtx.RoutePattern()
				spanName = r.Method + " " + routePattern
			}

			tp := otel.GetTracerProvider()
			tracer := tp.Tracer("go.opentelemetry.io/otel/trace")

			newHandler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cid := correlation.FromContext(r.Context())
				commonAttrs := []attribute.KeyValue{
					attribute.String(CorrelationIdKey, cid),
				}

				ctx, span := tracer.Start(
					r.Context(),
					spanName,
					trace.WithAttributes(commonAttrs...),
					trace.WithAttributes(semconv.HTTPClientAttributesFromHTTPRequest(r)...),
					trace.WithSpanKind(trace.SpanKindServer),
				)
				defer span.End()

				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
			}), routePattern)

			newHandler.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// OpenTelemtryMeterMiddleware is a middleware that adds open telemetry metrics for each http request.
// It counts requests per endpoint and measures the time taken to process the request.
func OpenTelemtryMeterMiddleware(chiRouter *chi.Mux) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			rCtx := chi.NewRouteContext()
			routePattern := ""
			if chiRouter.Match(rCtx, r.Method, r.URL.Path) {
				routePattern = rCtx.RoutePattern()
			}

			mp := otel.GetMeterProvider()
			meter := mp.Meter("go.opentelemetry.io/otel/metric")

			newHandler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rw := newStatusCodeCapturerWriter(w)
				counter, _ := meter.Int64Counter("http.request.count")
				counter.Add(r.Context(), 1, metric.WithAttributes([]attribute.KeyValue{
					attribute.String(string(semconv.HTTPRouteKey), routePattern),
					attribute.String(string(semconv.HTTPMethodKey), r.Method),
				}...))

				cid := correlation.FromContext(r.Context())

				initialTime := time.Now()
				next.ServeHTTP(rw, r)
				duration := time.Since(initialTime)

				commonAttrs := []attribute.KeyValue{
					attribute.String(CorrelationIdKey, cid),
					attribute.String(string(semconv.HTTPRouteKey), routePattern),
					attribute.String(string(semconv.HTTPMethodKey), r.Method),
					attribute.Int(string(semconv.HTTPStatusCodeKey), rw.statusCode),
				}

				durationHistogram, _ := meter.Int64Histogram("http.server.latency", metric.WithUnit("ms"))
				durationHistogram.Record(
					r.Context(),
					duration.Milliseconds(),
					metric.WithAttributes(commonAttrs...),
				)
			}), routePattern)

			newHandler.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// NewStatusCodeCapturerWriter creates an HTTP.ResponseWriter capable of
// capture the HTTP response status code.
func newStatusCodeCapturerWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
