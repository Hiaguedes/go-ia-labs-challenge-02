package domain

import "strings"

func CepRules(dirtyCep string) bool {
	cep := CleanCep(dirtyCep)
	
	if len(cep) != 8 {
		return false
	}
	return true
}

func CleanCep(dirtyCep string) string {
	cep := strings.TrimSpace(dirtyCep)
	cep = strings.Replace(cep, "-", "", -1)
	cep = strings.Replace(cep, ".", "", -1)
	return cep
}