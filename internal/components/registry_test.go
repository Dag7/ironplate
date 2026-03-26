package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	comp := Get("kafka")
	require.NotNil(t, comp)
	assert.Equal(t, "kafka", comp.Name)

	assert.Nil(t, Get("unknown"))
}

func TestList(t *testing.T) {
	names := List()
	assert.True(t, len(names) > 0)

	// Should be sorted
	for i := 1; i < len(names); i++ {
		assert.True(t, names[i-1] < names[i], "expected sorted order: %s < %s", names[i-1], names[i])
	}
}

func TestResolveDependencies(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		wantLen   int
		wantFirst string
		wantErr   bool
	}{
		{
			name:      "no dependencies",
			requested: []string{"kafka"},
			wantLen:   1,
			wantFirst: "kafka",
		},
		{
			name:      "argocd pulls in external-secrets",
			requested: []string{"argocd"},
			wantLen:   2,
			wantFirst: "external-secrets", // tier -1 comes before tier 3
		},
		{
			name:      "langfuse pulls in redis",
			requested: []string{"langfuse"},
			wantLen:   2,
			wantFirst: "redis", // tier 0 before tier 2
		},
		{
			name:    "unknown component",
			requested: []string{"unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveDependencies(tt.requested)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLen, len(resolved))
			if tt.wantFirst != "" {
				assert.Equal(t, tt.wantFirst, resolved[0])
			}
		})
	}
}

func TestSkillsForComponents(t *testing.T) {
	skills := SkillsForComponents([]string{"kafka", "redis"})
	assert.Contains(t, skills, "new-realtime-event")
	assert.Contains(t, skills, "setup-cache")
}

func TestClaudeMDSections(t *testing.T) {
	sections := ClaudeMDSections([]string{"hasura", "redis"})
	assert.Contains(t, sections, "graphql")
	assert.Contains(t, sections, "caching")
}
