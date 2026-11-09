package domain

import (
	"strconv"
)

func CepRules(dirtyCep string) bool {
	_, err := strconv.Atoi(dirtyCep)
	if err != nil || len(dirtyCep) != 8 {
		return false
	}
	
	return true
}