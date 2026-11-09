package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/usecase"
)

type WeatherHandler struct {
	ApiKey string
}

func NewWeatherHandler(apiKey string) *WeatherHandler {
	return &WeatherHandler{
		ApiKey: apiKey,
	}
}

func (h *WeatherHandler) WeatherHandler(w http.ResponseWriter, r *http.Request) {
	weatherHandler := usecase.NewGetWeatherByCepUseCase()
	cep := r.URL.Query().Get("cep")
	output, err := weatherHandler.Execute(cep, h.ApiKey)

	if err != nil {

		if err.Error() == "Invalid Zip Code" {
			http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
			return
		}

		if err.Error() == "can not find zipcode" {
			http.Error(w, "can not find zipcode", http.StatusNotFound)
			return
		}

		log.Println("Error fetching weather data: ", err)
		http.Error(w, "Error fetching weather data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(output)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
