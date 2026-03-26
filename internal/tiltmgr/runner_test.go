package tiltmgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTiltArgs(t *testing.T) {
	tests := []struct {
		name      string
		profile   *Profile
		noBrowser bool
		expected  []string
	}{
		{
			name: "basic profile",
			profile: &Profile{
				Services: []string{"auth-service"},
				Infra:    []string{"postgresql"},
			},
			expected: []string{"up", "--only", "postgresql", "--only", "auth-service"},
		},
		{
			name: "no browser",
			profile: &Profile{
				Services: []string{"api"},
				Infra:    []string{"redis"},
			},
			noBrowser: true,
			expected:  []string{"up", "--only", "redis", "--only", "api", "--no-browser"},
		},
		{
			name:     "empty profile",
			profile:  &Profile{},
			expected: []string{"up"},
		},
		{
			name: "infra only",
			profile: &Profile{
				Infra: []string{"postgresql", "redis", "kafka"},
			},
			expected: []string{"up", "--only", "postgresql", "--only", "redis", "--only", "kafka"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := buildTiltArgs(tc.profile, tc.noBrowser)
			assert.Equal(t, tc.expected, args)
		})
	}
}

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
		{"error", "error"},
		{"failed", "error"},
		{"CrashLoopBackOff", "error"},
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
