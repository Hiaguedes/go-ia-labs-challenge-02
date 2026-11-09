package main

import (
	"fmt"
	"log"

	config "github.com/hiaguedes/go-ia-labs-challenge-02/configs"
	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/web"
	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/webserver"
)

func main() {

	cfg, err := config.LoadWeatherConfig(".")
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
	}

	ws := webserver.NewWebServer(cfg.WebServerWeatherPort)
	weatherHandler := web.NewWeatherHandler(cfg.WeatherApiKey)
	ws.AddHandler("/", weatherHandler.WeatherHandler)
	fmt.Println("server started at " + cfg.WebServerWeatherPort + "...")
	ws.Start()
}