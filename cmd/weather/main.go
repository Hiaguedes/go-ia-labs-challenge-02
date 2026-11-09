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

	cfg, err := config.LoadWeatherConfig(".")
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
	}

	// Spans are exported to the otel-collector (OTEL_EXPORTER_OTLP_ENDPOINT, host:port).
	// A failure only disables tracing; the service keeps serving.
	if _, err := tracing.Setup(context.Background(), "service-weather"); err != nil {
		log.Printf("tracing disabled: %v", err)
	}

	ws := webserver.NewWebServer(":" + cfg.WebServerWeatherPort)
	weatherHandler := web.NewWeatherHandler(cfg.WeatherApiKey)
	ws.AddHandler("/", weatherHandler.WeatherHandler)
	// Prometheus scrape endpoint: Go runtime/process metrics by default.
	ws.AddHandler("/metrics", promhttp.Handler().ServeHTTP)
	fmt.Println("server started at " + cfg.WebServerWeatherPort + "...")
	ws.Start()
}