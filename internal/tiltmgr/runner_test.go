package tiltmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ok", "ok"},
		{"running", "ok"},
		{"Running", "ok"},
		{"pending", "pending"},
		{"building", "pending"},
		{"in_progress", "pending"},
		{"error", "error"},
		{"failed", "error"},
		{"CrashLoopBackOff", "error"},
		{"crashloopbackoff", "error"},
		{"", "unknown"},
		{"waiting", "waiting"},
		{"disabled", "disabled"},
		{"Disabled", "disabled"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeStatus(tc.input))
		})
	}
}
