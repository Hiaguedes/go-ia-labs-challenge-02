package main

import (
	"fmt"
	"log"

	config "github.com/hiaguedes/go-ia-labs-challenge-02/configs"
	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/web"
	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/infra/webserver"
)

func main() {

	cfg, err := config.LoadCepConfig(".")
	if err != nil {
		log.Fatal("Error loading configuration: ", err)
	}

	ws := webserver.NewWebServer(":" + cfg.WebServerCepPort)
	cepHandler := web.NewHandler()
	ws.AddHandler("/", cepHandler.CepValidatorHandler)
	fmt.Println("server started at " + cfg.WebServerCepPort + "...")
	ws.Start()
}