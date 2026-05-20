package graph

import (
	"strings"
	"fmt"
)

const (
	HCL_PATH_DELIMITER      = "."
	MAX_EXPECTED_PATH_PARTS = 3
)

// Joins HCL address parts avoiding accidental path separators
func tfJoin(parts ...string) string {
	filtered := make([]string, 0, MAX_EXPECTED_PATH_PARTS)
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, HCL_PATH_DELIMITER)
}

// Gets module name and its parent address
// TODO check how it works with count/for_each modules
func getModuleParts(addr string) (string, string, error) {
	parts := strings.Split(addr, HCL_PATH_DELIMITER)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("Could not get name from %v", addr)
	}

	name := parts[len(parts)-1]

	if len(parts) == 2 {
		return name, "", nil
	}

	parent := strings.Join(parts[:len(parts)-2], HCL_PATH_DELIMITER)

	return name, parent, nil
}

