package utils

import (
	"fmt"
	"strings"
	"time"
)

var timeLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02",
	time.RFC3339,
}

func ParseOptionalTime(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	for _, layout := range timeLayouts {
		parsed, err := time.ParseInLocation(layout, trimmed, time.Local)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid time format: %s", trimmed)
}
