cep-run:
	go run cmd/cep/main.go
weather-run:
	go run cmd/weather/main.go
docker-run:
	docker compose up --build -d