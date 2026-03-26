package devtools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferServiceName(t *testing.T) {
	tests := []struct {
		podName  string
		expected string
	}{
		{"auth-service-7f8d9c-x2k4", "auth-service"},
		{"api-gateway-5b4c3a-y1z2", "api-gateway"},
		{"simple-abc123", "simple-abc123"},
		{"a", "a"},
		{"billing-payment-service-abc-def", "billing-payment-service"},
	}

	for _, tc := range tests {
		t.Run(tc.podName, func(t *testing.T) {
			assert.Equal(t, tc.expected, inferServiceName(tc.podName))
		})
	}
}

func TestGroupPodsByService(t *testing.T) {
	pods := []Pod{
		{Name: "auth-service-abc-123", Status: "Running"},
		{Name: "auth-service-abc-456", Status: "Running"},
		{Name: "api-service-def-789", Status: "Pending"},
	}

	groups := GroupPodsByService(pods)

	assert.Len(t, groups, 2)

	// Groups are sorted alphabetically
	assert.Equal(t, "api-service", groups[0].Name)
	assert.Len(t, groups[0].Pods, 1)

	assert.Equal(t, "auth-service", groups[1].Name)
	assert.Len(t, groups[1].Pods, 2)
}

func TestStatusIcon(t *testing.T) {
	assert.Equal(t, "●", StatusIcon("Running"))
	assert.Equal(t, "◐", StatusIcon("Pending"))
	assert.Equal(t, "✓", StatusIcon("Succeeded"))
	assert.Equal(t, "✗", StatusIcon("Failed"))
	assert.Equal(t, "↻", StatusIcon("CrashLoopBackOff"))
	assert.Equal(t, "○", StatusIcon("Unknown"))
}
