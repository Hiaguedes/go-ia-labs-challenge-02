package usecase

import (
	"fmt"

	"github.com/hiaguedes/go-ia-labs-challenge-02/internal/domain"
)

type CEPValidatorOutputDTO struct{
	Cep string `json:"cep"`
}

type CepValidatorUseCase struct {
} 

func NewCepValidatorUseCase() *CepValidatorUseCase {
	return &CepValidatorUseCase{}
}

func (u *CepValidatorUseCase) Execute(cep string) (CEPValidatorOutputDTO, error){
	
	if !domain.CepRules(cep) {
		return CEPValidatorOutputDTO{}, fmt.Errorf("Invalid Zip Code")
	}

	return CEPValidatorOutputDTO{Cep: cep}, nil
}