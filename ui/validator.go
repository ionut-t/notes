package ui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
)

func validateNoteName(input *huh.Input) (string, error) {
	value := input.GetValue().(string)
	value = strings.Trim(value, " ")

	if value == "" {
		return "", errors.New("name cannot be empty")
	}

	if len(value) > 20 {
		return "", errors.New("name cannot be longer than 20 characters")
	}

	return value, nil
}
