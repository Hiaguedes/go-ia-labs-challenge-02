// Command metrics drives the CEP -> weather flow and records OpenTelemetry
// spans and metrics for it, so each hop can be inspected in Jaeger/Zipkin
// (traces) and Prometheus (metrics).
//
// The flow is:
//
//	cep-weather.flow (root, client)
//	└── cep.request        POST <CEP_SERVICE_URL>  {"cep": "..."}
//	    └── HTTP POST [service-cep]        (server span in the cep service)
//	        └── weather.request            (client span in the cep service)
//	            └── HTTP GET [service-weather]   (server span in the weather service)
//
// The request id is generated per run and sent in the X-Request-ID header, and
// the W3C traceparent header ties every span into the same trace.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hiaguedes/go-ia-labs-challenge-02/metrics/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// requestIDHeader is shared by every hop of a flow so the services and the
// traces can be correlated per request.
const requestIDHeader = "X-Request-ID"

const (
	defaultCepServiceURL = "http://localhost:8000"
	defaultCep           = "29193530"
	tracerName           = "github.com/hiaguedes/go-ia-labs-challenge-02/metrics"
)

// cepRequest is the JSON body expected by the cep service on POST.
type cepRequest struct {
	Cep string `json:"cep"`
}

// durationBuckets are explicit histogram boundaries in seconds, sized for
// HTTP latencies from a few milliseconds up to several seconds.
var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// instruments holds the metrics exported to Prometheus through the collector.
type instruments struct {
	flowDuration metric.Float64Histogram // whole cep -> weather round trip
	stepDuration metric.Float64Histogram // each HTTP hop, labelled by step
}

func newInstruments() (instruments, error) {
	meter := otel.Meter(tracerName)
	flow, err := meter.Float64Histogram("cep_weather_flow_duration_seconds",
		metric.WithDescription("Duration of the full CEP -> weather flow"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(durationBuckets...),
	)
	if err != nil {
		return instruments{}, fmt.Errorf("creating flow histogram: %w", err)
	}
	step, err := meter.Float64Histogram("cep_weather_step_duration_seconds",
		metric.WithDescription("Duration of each HTTP hop of the flow"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(durationBuckets...),
	)
	if err != nil {
		return instruments{}, fmt.Errorf("creating step histogram: %w", err)
	}
	return instruments{flowDuration: flow, stepDuration: step}, nil
}

type config struct {
	cepURL string
	cep    string
	runs   int
}

func main() {
	os.Exit(run())
}

// run holds the program logic and returns the process exit code. It is kept
// separate from main so deferred span flushing runs before the process exits;
// os.Exit skips deferred calls.
func run() int {
	cfg := config{}
	flag.StringVar(&cfg.cep, "cep", envOr("CEP", defaultCep), "CEP to query (env CEP)")
	flag.IntVar(&cfg.runs, "runs", 1, "number of times the flow is executed")
	flag.StringVar(&cfg.cepURL, "cep-url", envOr("CEP_SERVICE_URL", defaultCepServiceURL), "base URL of the cep service (env CEP_SERVICE_URL)")
	flag.Parse()

	if cfg.runs < 1 {
		log.Print("-runs must be at least 1")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdown, err := tracing.Setup(ctx, "metrics-client")
	if err != nil {
		log.Printf("setting up tracing: %v", err)
		return 1
	}
	defer func() {
		flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(flushCtx); err != nil {
			log.Printf("flushing spans: %v", err)
		}
	}()

	inst, err := newInstruments()
	if err != nil {
		log.Printf("setting up metrics: %v", err)
		return 1
	}

	client := &http.Client{Timeout: 30 * time.Second}
	tracer := otel.Tracer(tracerName)

	failures := 0
	for i := 1; i <= cfg.runs; i++ {
		if ctx.Err() != nil {
			break
		}
		if err := runFlow(ctx, tracer, inst, client, cfg, i); err != nil {
			failures++
			log.Printf("run %d failed: %v", i, err)
		}
	}

	if failures > 0 {
		log.Printf("%d of %d runs failed", failures, cfg.runs)
		return 1
	}
	return 0
}

// runFlow executes one CEP -> weather round trip under a single root span.
func runFlow(ctx context.Context, tracer trace.Tracer, inst instruments, client *http.Client, cfg config, run int) error {
	requestID := newRequestID()
	ctx, root := tracer.Start(ctx, "cep-weather.flow",
		trace.WithAttributes(
			attribute.String("cep.input", cfg.cep),
			attribute.Int("run", run),
			attribute.String("request.id", requestID),
		),
	)
	defer root.End()

	start := time.Now()
	// The flow histogram records every run, successful or not, so failures
	// show up in latency percentiles instead of being silently dropped.
	flowStatus := "ok"
	defer func() {
		inst.flowDuration.Record(ctx, time.Since(start).Seconds(),
			metric.WithAttributes(attribute.String("status", flowStatus)))
	}()

	body, err := call(ctx, tracer, inst, client, requestID, "cep.request", "cep", http.MethodPost, cfg.cepURL, cfg.cep)
	if err != nil {
		flowStatus = "error"
		return fail(root, err)
	}

	elapsed := time.Since(start)
	root.SetAttributes(attribute.Float64("flow.duration_ms", float64(elapsed.Microseconds())/1000))
	root.SetStatus(codes.Ok, "")

	log.Printf("run %d ok in %s: request_id=%s response=%s", run, elapsed.Round(time.Millisecond), requestID, string(body))
	return nil
}

// call sends the cep to baseURL inside a client span, injecting the trace
// context and the request id into the outgoing request headers. GET sends the
// cep as the `cep` query parameter; POST sends it as a JSON body {"cep": ...}.
// Any non-2xx status is returned as an error and marked on the span.
func call(ctx context.Context, tracer trace.Tracer, inst instruments, client *http.Client, requestID, spanName, step, method, baseURL, cep string) ([]byte, error) {
	ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	status := "error"
	start := time.Now()
	defer func() {
		inst.stepDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
			attribute.String("step", step),
			attribute.String("status", status),
		))
	}()

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fail(span, fmt.Errorf("parsing %s: %w", baseURL, err))
	}

	var reqBody io.Reader
	switch method {
	case http.MethodGet:
		q := u.Query()
		q.Set("cep", cep)
		u.RawQuery = q.Encode()
	case http.MethodPost:
		payload, err := json.Marshal(cepRequest{Cep: cep})
		if err != nil {
			return nil, fail(span, fmt.Errorf("encoding request body: %w", err))
		}
		reqBody = bytes.NewReader(payload)
	default:
		return nil, fail(span, fmt.Errorf("unsupported method %s", method))
	}

	span.SetAttributes(
		attribute.String("http.request.method", method),
		attribute.String("url.full", u.String()),
		attribute.String("request.id", requestID),
	)

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, fail(span, err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(requestIDHeader, requestID)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fail(span, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	elapsed := time.Since(start)
	span.SetAttributes(
		attribute.Int("http.response.status_code", resp.StatusCode),
		attribute.Float64("http.client.duration_ms", float64(elapsed.Microseconds())/1000),
	)
	if err != nil {
		return nil, fail(span, fmt.Errorf("reading response body: %w", err))
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fail(span, fmt.Errorf("%s returned %d: %s", spanName, resp.StatusCode, string(body)))
	}

	status = "ok"
	span.SetStatus(codes.Ok, "")
	return body, nil
}

// fail marks the span as errored and returns the error for the caller.
func fail(span trace.Span, err error) error {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

// newRequestID returns a random 128-bit hex id.
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
