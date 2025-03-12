package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

const maxNoteNameLength = 40

func validateNoteName(input *huh.Input) (string, error) {
	value := input.GetValue().(string)
	value = strings.Trim(value, " ")

	if value == "" {
		return "", errors.New("name cannot be empty")
	}

	if len(value) > maxNoteNameLength {
		return "", fmt.Errorf("name cannot be longer than %d characters", maxNoteNameLength)
	}

	return value, nil
}
