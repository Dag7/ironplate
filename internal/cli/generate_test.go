package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/engine"
	"github.com/dag7/ironplate/internal/scaffold"
)

func TestServiceTemplateDirs(t *testing.T) {
	expectedKeys := []string{"node-api", "go-api", "nextjs"}

	require.Len(t, serviceTemplateDirs, len(expectedKeys),
		"serviceTemplateDirs should have exactly %d entries", len(expectedKeys))

	tests := []struct {
		serviceType string
		expectedDir string
	}{
		{"node-api", "service/node"},
		{"go-api", "service/go"},
		{"nextjs", "service/nextjs"},
	}

	for _, tt := range tests {
		t.Run(tt.serviceType, func(t *testing.T) {
			dir, ok := serviceTemplateDirs[tt.serviceType]
			require.True(t, ok, "expected key %q to exist in serviceTemplateDirs", tt.serviceType)
			assert.Equal(t, tt.expectedDir, dir)
		})
	}
}

func TestRegisterServiceInRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal registry file
	regDir := filepath.Join(tmpDir, "tilt")
	err := os.MkdirAll(regDir, 0o755)
	require.NoError(t, err)

	regContent := `services: {}
infrastructure: {}
`
	err = os.WriteFile(filepath.Join(regDir, "registry.yaml"), []byte(regContent), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme", "acme.dev")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:      "auth-service",
		Type:      "node-api",
		Group:     "auth",
		Port:      3001,
		DebugPort: 9229,
		SrcFolder: "apps",
		Features:  []string{"hasura", "cache"},
	}

	err = scaffold.RegisterServiceInRegistry(tmpDir, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(regDir, "registry.yaml"))
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "auth-service:")
	assert.Contains(t, result, "type: node-api")
	assert.Contains(t, result, "group: auth")
	assert.Contains(t, result, "port: 3001")
	assert.Contains(t, result, "src: apps/auth-service")
	// Features should map to infra deps
	assert.Contains(t, result, "hasura")
	assert.Contains(t, result, "redis") // from "cache" feature
}

func TestRegisterServiceInRegistry_NoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()

	regDir := filepath.Join(tmpDir, "tilt")
	err := os.MkdirAll(regDir, 0o755)
	require.NoError(t, err)

	// Registry already has the service
	regContent := `services:
    auth-service:
        type: node-api
        group: auth
        port: 3001
        src: apps/auth-service
infrastructure: {}
`
	err = os.WriteFile(filepath.Join(regDir, "registry.yaml"), []byte(regContent), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme", "acme.dev")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:      "auth-service",
		Type:      "node-api",
		Group:     "auth",
		Port:      3001,
		SrcFolder: "apps",
	}

	// Should be idempotent — no error, no duplicate
	err = scaffold.RegisterServiceInRegistry(tmpDir, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(regDir, "registry.yaml"))
	require.NoError(t, err)

	result := string(data)
	count := 0
	for i := 0; i < len(result); {
		idx := findSubstring(result[i:], "auth-service:")
		if idx == -1 {
			break
		}
		count++
		i += idx + 1
	}
	assert.Equal(t, 1, count, "should not duplicate service in registry")
}

func TestRegisterServiceInRegistry_SecondService(t *testing.T) {
	tmpDir := t.TempDir()

	regDir := filepath.Join(tmpDir, "tilt")
	err := os.MkdirAll(regDir, 0o755)
	require.NoError(t, err)

	// Registry already has the first service
	regContent := `services:
    auth-service:
        type: node-api
        group: auth
        port: 3001
        src: apps/auth-service
infrastructure: {}
`
	err = os.WriteFile(filepath.Join(regDir, "registry.yaml"), []byte(regContent), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme", "acme.dev")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:      "auth-worker",
		Type:      "go-api",
		Group:     "auth",
		Port:      3002,
		DebugPort: 9230,
		SrcFolder: "apps",
	}

	err = scaffold.RegisterServiceInRegistry(tmpDir, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(regDir, "registry.yaml"))
	require.NoError(t, err)

	result := string(data)
	// Both services should be present
	assert.Contains(t, result, "auth-service:")
	assert.Contains(t, result, "auth-worker:")
	assert.Contains(t, result, "type: go-api")
}

func TestRegisterServiceInRegistry_CreatesNew(t *testing.T) {
	tmpDir := t.TempDir()

	// No tilt/ directory or registry file exists
	cfg := config.NewDefaultConfig("my-platform", "acme", "acme.dev")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:      "web-app",
		Type:      "nextjs",
		Group:     "frontend",
		Port:      3100,
		SrcFolder: "apps",
	}

	err := scaffold.RegisterServiceInRegistry(tmpDir, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "tilt", "registry.yaml"))
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "web-app:")
	assert.Contains(t, result, "type: nextjs")
	assert.Contains(t, result, "group: frontend")
	// nextjs should get "frontend" label instead of "backend"
	assert.Contains(t, result, "frontend")
}

func TestAppendServiceToUmbrellaValues(t *testing.T) {
	tmpDir := t.TempDir()

	valuesContent := `services:
  auth-service:
    enabled: true
    port: 3001
    image:
      tag: latest
    env:
      LOG_LEVEL: debug
`
	err := os.WriteFile(filepath.Join(tmpDir, "values.yaml"), []byte(valuesContent), 0o644)
	require.NoError(t, err)

	svc := &engine.ServiceTemplateData{
		Name: "auth-worker",
		Port: 3002,
	}

	err = scaffold.AppendServiceToUmbrellaValues(tmpDir, svc)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "values.yaml"))
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "  auth-worker:")
	assert.Contains(t, result, "    port: 3002")
	assert.Contains(t, result, "    enabled: true")
	// Original content preserved
	assert.Contains(t, result, "  auth-service:")
}

func TestAppendServiceToUmbrellaValues_NoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()

	valuesContent := `services:
  auth-service:
    enabled: true
    port: 3001
`
	err := os.WriteFile(filepath.Join(tmpDir, "values.yaml"), []byte(valuesContent), 0o644)
	require.NoError(t, err)

	svc := &engine.ServiceTemplateData{
		Name: "auth-service",
		Port: 3001,
	}

	err = scaffold.AppendServiceToUmbrellaValues(tmpDir, svc)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "values.yaml"))
	require.NoError(t, err)

	// Content should be unchanged — service already exists
	assert.Equal(t, valuesContent, string(data))
}

func TestRegisterArgoCDService_NewGroup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the ArgoCD values directory and file
	argoDir := filepath.Join(tmpDir, "k8s", "argocd", "charts", "apps")
	err := os.MkdirAll(argoDir, 0o755)
	require.NoError(t, err)

	valuesContent := `repoURL: https://github.com/acme/my-platform.git
targetRevision: HEAD
serviceGroups: {}
`
	err = os.WriteFile(filepath.Join(argoDir, "values.yaml"), []byte(valuesContent), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme", "acme.dev")
	cfg.Spec.GitOps.Enabled = true
	cfg.Spec.Infrastructure.Components = append(cfg.Spec.Infrastructure.Components, "argocd")

	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:  "auth-service",
		Group: "auth",
		Port:  3001,
	}

	err = scaffold.RegisterArgoCDService(tmpDir, cfg, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(argoDir, "values.yaml"))
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "auth:")
	assert.Contains(t, result, "syncWave:")
	assert.Contains(t, result, "chartPath: auth")
	assert.Contains(t, result, "name: auth-service")
}

func TestRegisterArgoCDService_ExistingGroup(t *testing.T) {
	tmpDir := t.TempDir()

	argoDir := filepath.Join(tmpDir, "k8s", "argocd", "charts", "apps")
	err := os.MkdirAll(argoDir, 0o755)
	require.NoError(t, err)

	valuesContent := `repoURL: https://github.com/acme/my-platform.git
serviceGroups:
    auth:
        syncWave: "4"
        chartPath: auth
        services:
            - name: auth-service
`
	err = os.WriteFile(filepath.Join(argoDir, "values.yaml"), []byte(valuesContent), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme", "acme.dev")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:  "auth-worker",
		Group: "auth",
		Port:  3002,
	}

	err = scaffold.RegisterArgoCDService(tmpDir, cfg, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(argoDir, "values.yaml"))
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "name: auth-service")
	assert.Contains(t, result, "name: auth-worker")
}

func TestRegisterArgoCDService_NoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()

	argoDir := filepath.Join(tmpDir, "k8s", "argocd", "charts", "apps")
	err := os.MkdirAll(argoDir, 0o755)
	require.NoError(t, err)

	valuesContent := `repoURL: https://github.com/acme/my-platform.git
serviceGroups:
    auth:
        syncWave: "4"
        chartPath: auth
        services:
            - name: auth-service
`
	err = os.WriteFile(filepath.Join(argoDir, "values.yaml"), []byte(valuesContent), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme", "acme.dev")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:  "auth-service",
		Group: "auth",
		Port:  3001,
	}

	err = scaffold.RegisterArgoCDService(tmpDir, cfg, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(argoDir, "values.yaml"))
	require.NoError(t, err)

	// Count occurrences of "auth-service"
	result := string(data)
	count := 0
	for i := 0; i < len(result); {
		idx := findSubstring(result[i:], "name: auth-service")
		if idx == -1 {
			break
		}
		count++
		i += idx + 1
	}
	assert.Equal(t, 1, count, "should not duplicate service in ArgoCD values")
}

func findSubstring(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestUpdateGoWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	goWorkPath := filepath.Join(tmpDir, "go.work")

	content := `go 1.24

use (
	./apps/api
)
`
	err := os.WriteFile(goWorkPath, []byte(content), 0o644)
	require.NoError(t, err)

	err = updateGoWorkspace(tmpDir, "my-lib")
	require.NoError(t, err)

	data, err := os.ReadFile(goWorkPath)
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "./packages/go/my-lib")
	assert.Contains(t, result, "./apps/api") // Original entry preserved
}

func TestUpdateGoWorkspace_NoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	goWorkPath := filepath.Join(tmpDir, "go.work")

	content := `go 1.24

use (
	./apps/api
	./packages/go/my-lib
)
`
	err := os.WriteFile(goWorkPath, []byte(content), 0o644)
	require.NoError(t, err)

	err = updateGoWorkspace(tmpDir, "my-lib")
	require.NoError(t, err)

	data, err := os.ReadFile(goWorkPath)
	require.NoError(t, err)

	// Content should be unchanged
	assert.Equal(t, content, string(data))
}

func TestUpdateGoWorkspace_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not error when go.work doesn't exist
	err := updateGoWorkspace(tmpDir, "my-lib")
	require.NoError(t, err)
}
