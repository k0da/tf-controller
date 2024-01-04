package utils

import (
	"fmt"
	"strings"
)

func ParseRenamePattern(pattern string) (string, string, error) {
	oldKey := pattern
	newKey := pattern
	if strings.Contains(pattern, ":") {
		parts := strings.Split(pattern, ":")
		if len(parts) != 2 {
			err := fmt.Errorf("invalid rename pattern %q", pattern)
			return "", "", err
		}

		if parts[0] == "" {
			err := fmt.Errorf("invalid rename pattern old name: %q", pattern)
			return "", "", err
		}

		if parts[1] == "" {
			err := fmt.Errorf("invalid rename pattern new name: %q", pattern)
			return "", "", err
		}

		oldKey = parts[0]
		newKey = parts[1]
	}

	return oldKey, newKey, nil
}
