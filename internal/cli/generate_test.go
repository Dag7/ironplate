package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/internal/engine"
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

func TestServiceTiltBuilders(t *testing.T) {
	// Every service type in serviceTemplateDirs should have a Tilt builder
	for svcType := range serviceTemplateDirs {
		t.Run(svcType, func(t *testing.T) {
			builder, ok := serviceTiltBuilders[svcType]
			assert.True(t, ok, "expected Tilt builder for service type %q", svcType)
			assert.NotEmpty(t, builder[0], "builder function should not be empty")
			assert.NotEmpty(t, builder[1], "builder module should not be empty")
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"auth-service", "auth_service"},
		{"my-api", "my_api"},
		{"simple", "simple"},
		{"already_snake", "already_snake"},
		{"UPPER-CASE", "upper_case"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, toSnakeCase(tt.input))
		})
	}
}

func TestInjectTiltfileEntry(t *testing.T) {
	tmpDir := t.TempDir()
	tiltfilePath := filepath.Join(tmpDir, "Tiltfile")

	// Create a Tiltfile with markers
	content := `target = "development"

# ==============================================================================
# IRONPLATE:SERVICES:START
# ==============================================================================
# IRONPLATE:SERVICES:END
`
	err := os.WriteFile(tiltfilePath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:      "auth-service",
		Type:      "node-api",
		Group:     "auth",
		Port:      3001,
		SrcFolder: "apps",
	}

	err = injectTiltfileEntry(tiltfilePath, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(tiltfilePath)
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "setup_auth(target)")
	assert.Contains(t, result, "load('./k8s/helm/my-platform/auth/Tiltfile'")
	assert.Contains(t, result, "# auth group")
}

func TestInjectTiltfileEntry_NoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	tiltfilePath := filepath.Join(tmpDir, "Tiltfile")

	content := `# IRONPLATE:SERVICES:START
# auth group
load('./k8s/helm/my-platform/auth/Tiltfile', 'setup_auth')
setup_auth(target)

# IRONPLATE:SERVICES:END
`
	err := os.WriteFile(tiltfilePath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:      "auth-service",
		Type:      "node-api",
		Group:     "auth",
		Port:      3001,
		SrcFolder: "apps",
	}

	// Should be idempotent
	err = injectTiltfileEntry(tiltfilePath, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(tiltfilePath)
	require.NoError(t, err)

	// Count occurrences — should still be exactly 1
	result := string(data)
	count := 0
	for i := 0; i < len(result); {
		idx := findSubstring(result[i:], "setup_auth(target)")
		if idx == -1 {
			break
		}
		count++
		i += idx + 1
	}
	assert.Equal(t, 1, count, "should not duplicate Tiltfile entry")
}

func TestInjectTiltfileEntry_SecondServiceSameGroup(t *testing.T) {
	tmpDir := t.TempDir()
	tiltfilePath := filepath.Join(tmpDir, "Tiltfile")

	// Group already injected by first service
	content := `# IRONPLATE:SERVICES:START
# auth group
load('./k8s/helm/my-platform/auth/Tiltfile', 'setup_auth')
setup_auth(target)

# IRONPLATE:SERVICES:END
`
	err := os.WriteFile(tiltfilePath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:      "auth-worker",
		Type:      "go-api",
		Group:     "auth",
		Port:      3002,
		SrcFolder: "apps",
	}

	// Second service in same group — should NOT add another entry
	err = injectTiltfileEntry(tiltfilePath, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(tiltfilePath)
	require.NoError(t, err)

	result := string(data)
	count := 0
	for i := 0; i < len(result); {
		idx := findSubstring(result[i:], "setup_auth(target)")
		if idx == -1 {
			break
		}
		count++
		i += idx + 1
	}
	assert.Equal(t, 1, count, "should not add duplicate group entry for second service")
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

	err = appendServiceToUmbrellaValues(tmpDir, svc)
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

	err = appendServiceToUmbrellaValues(tmpDir, svc)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "values.yaml"))
	require.NoError(t, err)

	// Content should be unchanged — service already exists
	assert.Equal(t, valuesContent, string(data))
}

func TestRegisterArgoCDService_NewGroup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create the ArgoCD values directory and file
	argoDir := filepath.Join(tmpDir, "k8s", "helm", "infra", "argocd")
	err := os.MkdirAll(argoDir, 0o755)
	require.NoError(t, err)

	valuesContent := `repoURL: https://github.com/acme/my-platform.git
targetRevision: HEAD
serviceGroups: {}
`
	err = os.WriteFile(filepath.Join(argoDir, "values.yaml"), []byte(valuesContent), 0o644)
	require.NoError(t, err)

	cfg := config.NewDefaultConfig("my-platform", "acme")
	cfg.Spec.GitOps.Enabled = true
	cfg.Spec.Infrastructure.Components = append(cfg.Spec.Infrastructure.Components, "argocd")

	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:  "auth-service",
		Group: "auth",
		Port:  3001,
	}

	err = registerArgoCDService(tmpDir, cfg, ctx)
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

	argoDir := filepath.Join(tmpDir, "k8s", "helm", "infra", "argocd")
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

	cfg := config.NewDefaultConfig("my-platform", "acme")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:  "auth-worker",
		Group: "auth",
		Port:  3002,
	}

	err = registerArgoCDService(tmpDir, cfg, ctx)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(argoDir, "values.yaml"))
	require.NoError(t, err)

	result := string(data)
	assert.Contains(t, result, "name: auth-service")
	assert.Contains(t, result, "name: auth-worker")
}

func TestRegisterArgoCDService_NoDuplicate(t *testing.T) {
	tmpDir := t.TempDir()

	argoDir := filepath.Join(tmpDir, "k8s", "helm", "infra", "argocd")
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

	cfg := config.NewDefaultConfig("my-platform", "acme")
	ctx := engine.NewTemplateContext(cfg)
	ctx.Service = &engine.ServiceTemplateData{
		Name:  "auth-service",
		Group: "auth",
		Port:  3001,
	}

	err = registerArgoCDService(tmpDir, cfg, ctx)
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
