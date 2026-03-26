package engine

import (
	"testing"
	"time"

	"github.com/dag7/ironplate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig() *config.ProjectConfig {
	return &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "my-platform",
			Organization: "acme-corp",
			Domain:       "acme.dev",
			Description:  "Test project",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node", "go"},
			Monorepo: config.MonorepoSpec{
				PackageManager: "pnpm",
				NodeVersion:    "22",
				GoVersion:      "1.24",
				BuildSystem:    "nx",
				Scopes:         []string{"@my-platform", "@oss"},
			},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "hasura", "dapr", "redis", "observability", "external-secrets", "argocd", "langfuse"},
			},
		},
	}
}

func TestNewTemplateContext_BasicFields(t *testing.T) {
	cfg := newTestConfig()
	ctx := NewTemplateContext(cfg)

	require.NotNil(t, ctx)
	assert.Same(t, cfg, ctx.Project, "Project should reference the original config")
	assert.NotEmpty(t, ctx.GoModule)
	assert.NotEmpty(t, ctx.Computed.NamePascal)
}

func TestNewTemplateContext_ComputedValues(t *testing.T) {
	cfg := newTestConfig()
	ctx := NewTemplateContext(cfg)

	assert.True(t, ctx.Computed.HasNode, "HasNode should be true when 'node' is in languages")
	assert.True(t, ctx.Computed.HasGo, "HasGo should be true when 'go' is in languages")

	assert.Equal(t, "MyPlatform", ctx.Computed.NamePascal)
	assert.Equal(t, "myPlatform", ctx.Computed.NameCamel)
	assert.Equal(t, "my_platform", ctx.Computed.NameSnake)

	// Backward-compat aliases should match
	assert.Equal(t, ctx.Computed.NamePascal, ctx.Computed.ProjectNamePascal)
	assert.Equal(t, ctx.Computed.NameCamel, ctx.Computed.ProjectNameCamel)
	assert.Equal(t, ctx.Computed.NameSnake, ctx.Computed.ProjectNameSnake)
}

func TestNewTemplateContext_ComponentFlags(t *testing.T) {
	tests := []struct {
		name       string
		components []string
		check      func(t *testing.T, cv ComputedValues)
	}{
		{
			name:       "all components",
			components: []string{"kafka", "hasura", "dapr", "redis", "observability", "external-secrets", "argocd", "langfuse"},
			check: func(t *testing.T, cv ComputedValues) {
				assert.True(t, cv.HasKafka)
				assert.True(t, cv.HasHasura)
				assert.True(t, cv.HasDapr)
				assert.True(t, cv.HasRedis)
				assert.True(t, cv.HasObservability)
				assert.True(t, cv.HasExternalSecrets)
				assert.True(t, cv.HasArgoCD)
				assert.True(t, cv.HasLangfuse)
			},
		},
		{
			name:       "no components",
			components: []string{},
			check: func(t *testing.T, cv ComputedValues) {
				assert.False(t, cv.HasKafka)
				assert.False(t, cv.HasHasura)
				assert.False(t, cv.HasDapr)
				assert.False(t, cv.HasRedis)
				assert.False(t, cv.HasObservability)
				assert.False(t, cv.HasExternalSecrets)
				assert.False(t, cv.HasArgoCD)
				assert.False(t, cv.HasLangfuse)
			},
		},
		{
			name:       "partial components",
			components: []string{"kafka", "redis"},
			check: func(t *testing.T, cv ComputedValues) {
				assert.True(t, cv.HasKafka)
				assert.True(t, cv.HasRedis)
				assert.False(t, cv.HasHasura)
				assert.False(t, cv.HasDapr)
				assert.False(t, cv.HasObservability)
				assert.False(t, cv.HasExternalSecrets)
				assert.False(t, cv.HasArgoCD)
				assert.False(t, cv.HasLangfuse)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig()
			cfg.Spec.Infrastructure.Components = tt.components
			ctx := NewTemplateContext(cfg)
			tt.check(t, ctx.Computed)
		})
	}
}

func TestNewTemplateContext_PrimaryScope(t *testing.T) {
	t.Run("default scope from organization", func(t *testing.T) {
		cfg := newTestConfig()
		cfg.Spec.Monorepo.Scopes = nil
		ctx := NewTemplateContext(cfg)

		assert.Equal(t, "@acme-corp", ctx.Computed.PrimaryScope)
	})

	t.Run("override from Scopes", func(t *testing.T) {
		cfg := newTestConfig()
		cfg.Spec.Monorepo.Scopes = []string{"@my-platform", "@oss"}
		ctx := NewTemplateContext(cfg)

		assert.Equal(t, "@my-platform", ctx.Computed.PrimaryScope)
	})

	t.Run("empty scopes falls back to org", func(t *testing.T) {
		cfg := newTestConfig()
		cfg.Spec.Monorepo.Scopes = []string{}
		ctx := NewTemplateContext(cfg)

		assert.Equal(t, "@acme-corp", ctx.Computed.PrimaryScope)
	})
}

func TestNewTemplateContext_GoModule(t *testing.T) {
	cfg := newTestConfig()
	ctx := NewTemplateContext(cfg)

	assert.Equal(t, "github.com/acme-corp/my-platform", ctx.GoModule)
}

func TestNewTemplateContext_Year(t *testing.T) {
	cfg := newTestConfig()
	ctx := NewTemplateContext(cfg)

	assert.Equal(t, time.Now().Year(), ctx.Computed.Year)
}

func TestServiceTemplateData_HasFeature(t *testing.T) {
	svc := &ServiceTemplateData{
		Name:     "auth-service",
		Features: []string{"dapr", "eventbus", "cache"},
	}

	assert.True(t, svc.HasFeature("dapr"))
	assert.True(t, svc.HasFeature("eventbus"))
	assert.True(t, svc.HasFeature("cache"))
	assert.False(t, svc.HasFeature("hasura"))
	assert.False(t, svc.HasFeature(""))
}

func TestServiceTemplateData_HasFeature_Empty(t *testing.T) {
	svc := &ServiceTemplateData{
		Name:     "simple-service",
		Features: nil,
	}

	assert.False(t, svc.HasFeature("dapr"))
}
