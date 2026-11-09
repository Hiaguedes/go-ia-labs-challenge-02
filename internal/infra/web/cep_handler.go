package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/usecase"
)

type Handler struct {
	
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) CepValidatorHandler(w http.ResponseWriter, r *http.Request) {
	cepUseCase := usecase.NewCepValidatorUseCase()
	cep := r.URL.Query().Get("cep")
	output, err := cepUseCase.Execute(cep)

	if err != nil {

	
	if err.Error() == "Invalid Zip Code" {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
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