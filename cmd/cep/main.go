package main

import (
	"context"
	"fmt"
	"log"

	config "github.com/hiaguedes/go-ia-labs-challenge-02/configs"
	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/web"
	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/webserver"
	"github.com/hiaguedes/go-ia-labs-challenge-02/metrics/tracing"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	cfg, err := config.LoadCepConfig(".")
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
	}

	// Spans are exported to the otel-collector (OTEL_EXPORTER_OTLP_ENDPOINT, host:port).
	// A failure only disables tracing; the service keeps serving.
	if _, err := tracing.Setup(context.Background(), "service-cep"); err != nil {
		log.Printf("tracing disabled: %v", err)
	}

	ws := webserver.NewWebServer(":" + cfg.WebServerCepPort)
	cepHandler := web.NewHandler(cfg.WeatherServiceURL)
	ws.AddHandler("/", cepHandler.CepValidatorHandler)
	// Prometheus scrape endpoint: Go runtime/process metrics by default.
	ws.AddHandler("/metrics", promhttp.Handler().ServeHTTP)
	fmt.Println("server started at " + cfg.WebServerCepPort + "...")
	ws.Start()
}