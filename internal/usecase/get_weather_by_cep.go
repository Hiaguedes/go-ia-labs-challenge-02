package usecase

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/domain"
)

type CEPResponse struct {
	Cep         string `json:"cep"`
	Localidade  string `json:"localidade"`
}

type WeatherInputDTO struct {
	ApiKey string `json:"api_key"`
	Cep    string `json:"cep"`
}

type WeatherOutputDTO struct {
	City string `json:"city"`
	TempC string `json:"temp_C"`
	TempF string `json:"temp_F"`
	TempK string `json:"temp_K"`
}

type WeatherResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
		TempF float64 `json:"temp_f"`
	} `json:"current"`
}

type GetWeatherByCepUseCase struct {
}

func NewGetWeatherByCepUseCase() *GetWeatherByCepUseCase {
	return &GetWeatherByCepUseCase{}
}

func (u *GetWeatherByCepUseCase)  GetFullAddress(cep string) (CEPResponse, error) {
	req, err := http.Get("https://viacep.com.br/ws/" + cep + "/json/")
	if err != nil {
		return CEPResponse{}, err
	}
	defer req.Body.Close()

	var cepResponse CEPResponse
	if err = json.NewDecoder(req.Body).Decode(&cepResponse); err != nil {
		return CEPResponse{}, err
	}
	return cepResponse, nil
}

func (u *GetWeatherByCepUseCase) GetWeatherByCEP(city string, apiKey string) (WeatherResponse, error) {

	// Get the weather by city and state
	req, err := http.Get("https://api.weatherapi.com/v1/current.json?q=" + url.PathEscape(city) + "&key=" + apiKey)
	if err != nil {
		return WeatherResponse{}, err
	}
	defer req.Body.Close()

	var weatherResponse WeatherResponse
	if err = json.NewDecoder(req.Body).Decode(&weatherResponse); err != nil {
		return WeatherResponse{}, err
	}
	return weatherResponse, nil

}

func (u *GetWeatherByCepUseCase) Execute(cep string, apiKey string) (WeatherOutputDTO, error) {
	if !domain.CepRules(cep) {
		return WeatherOutputDTO{}, fmt.Errorf("Invalid Zip Code")
	}
	cepResponse, err := u.GetFullAddress(cep)
	// fmt.Printf("CEP Response: %v, localidade: %s\n", cepResponse, cepResponse.Localidade)
	if err != nil {
		return WeatherOutputDTO{}, err
	}

	if cepResponse.Localidade == "" {
		return WeatherOutputDTO{}, fmt.Errorf("can not find zipcode")
	}

	weatherResponse, err := u.GetWeatherByCEP(cepResponse.Localidade, apiKey)
	// fmt.Printf("Weather Response: %v\n", weatherResponse)
	if err != nil {
		return WeatherOutputDTO{}, err
	}

	dto := WeatherOutputDTO{
		City: cepResponse.Localidade,
		TempC: fmt.Sprintf("%.2f", weatherResponse.Current.TempC),
		TempF: fmt.Sprintf("%.2f", weatherResponse.Current.TempF),
		TempK: fmt.Sprintf("%.2f", weatherResponse.Current.TempC+273.15),
	}

	return dto, nil
}