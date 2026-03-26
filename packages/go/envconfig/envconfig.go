// Package envconfig provides utilities for loading typed configuration
// from environment variables with default values.
package envconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetString returns the value of the environment variable named by the key,
// or the fallback if the variable is not set or empty.
func GetString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// MustGetString returns the value of the environment variable named by the key.
// It panics if the variable is not set or empty.
func MustGetString(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("envconfig: required environment variable %q is not set", key))
	}
	return v
}

// GetInt returns the integer value of the environment variable named by the key,
// or the fallback if the variable is not set, empty, or not a valid integer.
func GetInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

// GetBool returns the boolean value of the environment variable named by the key,
// or the fallback if the variable is not set or not a valid boolean.
// Accepted truthy values: "1", "true", "yes", "on" (case-insensitive).
func GetBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// GetDuration returns the duration value of the environment variable named by the key,
// or the fallback if the variable is not set or not a valid duration.
func GetDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// GetStringSlice returns a string slice by splitting the environment variable
// value on the given separator, or the fallback if the variable is not set.
func GetStringSlice(key, sep string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
