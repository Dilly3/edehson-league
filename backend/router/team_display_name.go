package router

import (
	"errors"
	"strings"
	"unicode/utf8"
)

func normalizeTeamDisplayName(input *string) (*string, error) {
	if input == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*input)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > 12 {
		return nil, errors.New("Display name must contain at most 12 characters")
	}
	return &value, nil
}
