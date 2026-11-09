package webserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// RequestIDHeader carries the request id shared across the services of a flow.
// The caller sends it; if absent, the server generates one. It is echoed back
// in the response and recorded on the server span and in the access log.
const RequestIDHeader = "X-Request-ID"

// metricsPath is excluded from tracing and metrics so scrapes do not pollute them.
const metricsPath = "/metrics"

var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_server_requests_total",
		Help: "Total HTTP requests handled, by method, route and status code.",
	}, []string{"method", "route", "status_code"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_server_request_duration_seconds",
		Help:    "Duration of HTTP requests handled, by method, route and status code.",
		Buckets: durationBuckets,
	}, []string{"method", "route", "status_code"})
)

type requestIDKey struct{}

// RequestIDFromContext returns the request id attached by the telemetry middleware.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// statusRecorder captures the status code written by the downstream handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// telemetryMiddleware extracts the incoming W3C trace context, so the server
// span becomes a child of the caller's span, starts a server span, propagates
// the request id, and records Prometheus request metrics.
func telemetryMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer("github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/webserver")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == metricsPath {
			next.ServeHTTP(w, r)
			return
		}

		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set(RequestIDHeader, requestID)

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, "HTTP "+r.Method,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("request.id", requestID),
				attribute.String("http.request.method", r.Method),
				attribute.String("url.path", r.URL.Path),
			),
		)
		defer span.End()
		ctx = context.WithValue(ctx, requestIDKey{}, requestID)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r.WithContext(ctx))
		elapsed := time.Since(start)

		// chi resolves the route pattern while dispatching; it is read from the
		// same RouteContext the request carried in, so it is available here.
		route := "unmatched"
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.RoutePattern() != "" {
			route = rctx.RoutePattern()
		}
		status := strconv.Itoa(rec.status)

		httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route, status).Observe(elapsed.Seconds())

		span.SetAttributes(
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", rec.status),
		)
		if rec.status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(rec.status))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		log.Printf("request_id=%s method=%s route=%s status=%d duration_ms=%.2f",
			requestID, r.Method, route, rec.status, float64(elapsed.Microseconds())/1000)
	})
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
