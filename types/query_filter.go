package types

import (
	"strings"
)

func All(pattern string) bool {
	return true
}

func None(pattern string) bool {
	return false
}

// Get fromt the server connected
func Under(pattern string, target string) bool {
	return strings.Contains(target, pattern)
}
