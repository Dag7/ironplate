package devtools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLocalContextName(t *testing.T) {
	assert.Equal(t, "k3d-my-project-cluster", GetLocalContextName("my-project"))
	assert.Equal(t, "k3d-project-zero-cluster", GetLocalContextName("project-zero"))
}

func TestParseContextLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected KubeContext
	}{
		{
			name: "current context",
			line: "*  k3d-my-cluster   k3d-my-cluster   k3d-my-cluster   default",
			expected: KubeContext{
				Name:      "k3d-my-cluster",
				Cluster:   "k3d-my-cluster",
				Namespace: "default",
				Current:   true,
			},
		},
		{
			name: "non-current context",
			line: "   gke-staging   gke-cluster   gke-user   myapp",
			expected: KubeContext{
				Name:      "gke-staging",
				Cluster:   "gke-cluster",
				Namespace: "myapp",
				Current:   false,
			},
		},
		{
			name:     "minimal context",
			line:     "   context1   cluster1",
			expected: KubeContext{Name: "context1", Cluster: "cluster1"},
		},
		{
			name:     "empty line",
			line:     "",
			expected: KubeContext{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseContextLine(tc.line)
			assert.Equal(t, tc.expected.Name, result.Name)
			assert.Equal(t, tc.expected.Cluster, result.Cluster)
			assert.Equal(t, tc.expected.Current, result.Current)
			assert.Equal(t, tc.expected.Namespace, result.Namespace)
		})
	}
}
