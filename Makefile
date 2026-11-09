cep-run:
	go run cmd/cep/main.go
weather-run:
	go run cmd/weather/main.go
docker-run:
	docker compose up --build -d

# Tracing / manual testing (see metrics/)
# Start the observability stack (Jaeger UI: http://localhost:16686,
# Prometheus: http://localhost:9090, Grafana: http://localhost:3001):
#   make metrics-up
# Run cep and weather locally in other terminals (or use `make docker-run`):
#   make cep-run / make weather-run
# Execute the CEP -> weather flow with spans. Override CEP and RUNS as needed:
#   make metrics-run CEP=29193530 RUNS=3
CEP ?= 29193530
RUNS ?= 1

metrics-up:
	docker compose up -d --force-recreate jaeger-all-in-one zipkin otel-collector prometheus grafana

metrics-down:
	docker compose stop jaeger-all-in-one zipkin otel-collector prometheus grafana

metrics-run:
	go run ./metrics -cep $(CEP) -runs $(RUNS)