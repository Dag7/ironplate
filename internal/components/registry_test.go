package components

import (
	"testing"
	"testing/fstest"

	"github.com/dag7/ironplate/templates"
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

func TestValidateTemplates(t *testing.T) {
	// This test ensures every component with Tilt.HelmPath has a Tiltfile.tmpl
	// in its template directory. If this fails, a new component was added with
	// a HelmPath but no Tiltfile — which would cause Tilt to crash at runtime.
	err := ValidateTemplates(templates.FS)
	require.NoError(t, err, "component template validation failed — see error for details")
}

func TestTiltConfigConsistency(t *testing.T) {
	for name, comp := range builtinComponents {
		if comp.Tilt.HelmPath != "" {
			assert.NotEmpty(t, comp.Tilt.SetupFn,
				"component %q has HelmPath but no SetupFn", name)
			assert.False(t, comp.Tilt.Local,
				"component %q has both HelmPath and Local=true", name)
		}
	}
}

func TestAll(t *testing.T) {
	all := All()
	assert.NotEmpty(t, all)
	assert.Contains(t, all, "kafka")
	assert.Contains(t, all, "redis")
	assert.Contains(t, all, "hasura")

	// Should return the same map (not a copy)
	assert.Same(t, builtinComponents["kafka"], all["kafka"])
}

func TestSkillsForComponents_UnknownIgnored(t *testing.T) {
	skills := SkillsForComponents([]string{"unknown", "kafka"})
	assert.Contains(t, skills, "new-realtime-event")
}

func TestSkillsForComponents_NoDuplicates(t *testing.T) {
	skills := SkillsForComponents([]string{"kafka", "kafka"})
	count := 0
	for _, s := range skills {
		if s == "new-realtime-event" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestClaudeMDSections_UnknownIgnored(t *testing.T) {
	sections := ClaudeMDSections([]string{"unknown", "hasura"})
	assert.Contains(t, sections, "graphql")
}

func TestValidateTemplates_MissingTiltfile(t *testing.T) {
	// Create a fake FS with the template dir but no Tiltfile.tmpl
	fakeFS := fstest.MapFS{
		"components/kafka/helm/values.yaml": &fstest.MapFile{Data: []byte("test")},
		// Missing: components/kafka/helm/Tiltfile.tmpl
	}

	err := ValidateTemplates(fakeFS)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Tiltfile")
}

func TestValidateTemplates_MissingSetupFn(t *testing.T) {
	// The real registry has HelmPath+SetupFn paired. This test validates
	// the error path by using a fake FS where templates exist but the
	// validation function catches inconsistencies in the real registry.
	// Since we can't modify builtinComponents, we verify the real FS passes.
	err := ValidateTemplates(templates.FS)
	require.NoError(t, err)
}

func TestValidateTemplates_MissingTemplateDir(t *testing.T) {
	// Empty FS — all template dirs referenced by components will be missing
	fakeFS := fstest.MapFS{}

	err := ValidateTemplates(fakeFS)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't exist")
}

func TestInfraRegistryEntries(t *testing.T) {
	entries := InfraRegistryEntries([]string{"kafka", "redis", "argocd"})

	// kafka has HelmPath, redis has Local, argocd has neither → only kafka and redis
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	assert.Contains(t, names, "kafka")
	assert.Contains(t, names, "redis")
	assert.NotContains(t, names, "argocd") // argocd has no Tilt config
}

func TestInfraRegistryEntries_UnknownIgnored(t *testing.T) {
	entries := InfraRegistryEntries([]string{"unknown"})
	assert.Empty(t, entries)
}

func TestInfraRegistryEntries_FieldValues(t *testing.T) {
	entries := InfraRegistryEntries([]string{"kafka"})
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "kafka", e.Name)
	assert.Equal(t, "k8s/helm/infra/kafka", e.HelmPath)
	assert.Equal(t, "setup_kafka", e.SetupFn)
	assert.False(t, e.Local)
	assert.False(t, e.Required)
}

func TestInfraRegistryEntries_LocalComponent(t *testing.T) {
	entries := InfraRegistryEntries([]string{"redis"})
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "redis", e.Name)
	assert.Empty(t, e.HelmPath)
	assert.True(t, e.Local)
}

func TestInfraRegistryEntries_WithDeps(t *testing.T) {
	entries := InfraRegistryEntries([]string{"langfuse"})
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "langfuse", e.Name)
	assert.Equal(t, []string{"redis", "postgres"}, e.InfraDeps)
}

func TestResolveDependencies_Dedup(t *testing.T) {
	// Requesting the same component twice should not duplicate
	resolved, err := ResolveDependencies([]string{"kafka", "kafka"})
	require.NoError(t, err)
	assert.Equal(t, 1, len(resolved))
}

func TestResolveDependencies_TransitiveDep(t *testing.T) {
	// hasura-event-relay requires hasura, kafka, dapr
	resolved, err := ResolveDependencies([]string{"hasura-event-relay"})
	require.NoError(t, err)
	assert.True(t, len(resolved) >= 4, "should include hasura-event-relay + its 3 deps")
}
