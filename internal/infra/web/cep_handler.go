package web

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/webserver"
	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/usecase"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/web"

type CepValidatorBodyDTO struct {
	Cep string `json:"cep"`
}

// Handler serves the cep service (Service A). It validates the CEP received in
// the POST body and forwards it to the weather service (Service B).
type Handler struct {
	weatherServiceURL string
	client            *http.Client
}

func NewHandler(weatherServiceURL string) *Handler {
	return &Handler{
		weatherServiceURL: weatherServiceURL,
		client:            &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *Handler) CepValidatorHandler(w http.ResponseWriter, r *http.Request) {
	var body CepValidatorBodyDTO
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	cepUseCase := usecase.NewCepValidatorUseCase()
	output, err := cepUseCase.Execute(body.Cep)
	if err != nil {
		if err.Error() == "Invalid Zip Code" {
			http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
			return
		}
		log.Println("Error validating cep: ", err)
		http.Error(w, "Error validating cep", http.StatusInternalServerError)
		return
	}

	weatherBody, status, err := h.forwardToWeather(r.Context(), output.Cep)
	if err != nil {
		log.Println("Error calling weather service: ", err)
		http.Error(w, "Error fetching weather data", http.StatusBadGateway)
		return
	}

	if status != http.StatusOK {
		// Client errors from Service B (invalid or unknown zipcode) are passed
		// through; anything else is reported as a bad gateway.
		if status >= 400 && status < 500 {
			http.Error(w, strings.TrimSpace(string(weatherBody)), status)
			return
		}
		http.Error(w, "Error fetching weather data", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(weatherBody); err != nil {
		log.Println("Error writing response: ", err)
	}
}

// forwardToWeather calls Service B with the cep as the `cep` query parameter. It
// runs under a client span, propagating the trace context and the request id so
// Service B's server span joins the same trace.
func (h *Handler) forwardToWeather(ctx context.Context, cep string) ([]byte, int, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "weather.request", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	u, err := url.Parse(h.weatherServiceURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, err
	}
	q := u.Query()
	q.Set("cep", cep)
	u.RawQuery = q.Encode()

	requestID := webserver.RequestIDFromContext(ctx)
	span.SetAttributes(
		attribute.String("url.full", u.String()),
		attribute.String("request.id", requestID),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, err
	}
	if requestID != "" {
		req.Header.Set(webserver.RequestIDHeader, requestID)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := h.client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, 0, err
	}

	span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	if resp.StatusCode >= http.StatusInternalServerError {
		span.SetStatus(codes.Error, resp.Status)
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return body, resp.StatusCode, nil
}
