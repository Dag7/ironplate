//go:build integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/dag7/ironplate/internal/config"
)

// ============================================================================
// CLI Integration: iron init --non-interactive
// ============================================================================

func TestCLI_InitNonInteractive_NodeOnly(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "test-proj")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "test-proj",
		"--org", "testorg",
		"--domain", "test.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
	})

	err := root.Execute()
	require.NoError(t, err)

	// Verify project was created
	assertCLIFileExists(t, outputDir, "ironplate.yaml")
	assertCLIFileExists(t, outputDir, "package.json")
	assertCLIFileExists(t, outputDir, "README.md")

	// Verify config
	cfg := loadCLIConfig(t, outputDir)
	assert.Equal(t, "test-proj", cfg.Metadata.Name)
	assert.Equal(t, "testorg", cfg.Metadata.Organization)
	assert.Equal(t, []string{"node"}, cfg.Spec.Languages)
	assert.Equal(t, "none", cfg.Spec.Cloud.Provider)
}

func TestCLI_InitNonInteractive_GoOnly(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "go-proj")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "go-proj",
		"--org", "testorg",
		"--domain", "go.dev",
		"--language", "go",
		"--provider", "none",
		"--preset", "minimal",
	})

	err := root.Execute()
	require.NoError(t, err)

	assertCLIFileExists(t, outputDir, "go.work")
	assertCLIFileNotExists(t, outputDir, "package.json")

	cfg := loadCLIConfig(t, outputDir)
	assert.Equal(t, []string{"go"}, cfg.Spec.Languages)
}

func TestCLI_InitNonInteractive_FullStack(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "full-proj")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "full-proj",
		"--org", "acme",
		"--domain", "full.dev",
		"--language", "mixed",
		"--provider", "gcp",
		"--preset", "full",
	})

	err := root.Execute()
	require.NoError(t, err)

	assertCLIFileExists(t, outputDir, "package.json")
	assertCLIFileExists(t, outputDir, "go.work")
	assertCLIFileExists(t, outputDir, "k8s/helm/infra/kafka/Chart.yaml")
	assertCLIFileExists(t, outputDir, "k8s/helm/infra/redis/Chart.yaml")
	assertCLIFileExists(t, outputDir, "iac/pulumi/Pulumi.yaml")

	cfg := loadCLIConfig(t, outputDir)
	assert.Equal(t, []string{"node", "go"}, cfg.Spec.Languages)
	assert.Equal(t, "gcp", cfg.Spec.Cloud.Provider)
	assert.True(t, len(cfg.Spec.Infrastructure.Components) > 3)
}

func TestCLI_InitNonInteractive_WithExampleServices(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "svc-proj")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "svc-proj",
		"--org", "acme",
		"--domain", "svc.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
		"--example-services",
	})

	err := root.Execute()
	require.NoError(t, err)

	// Example services should be generated
	assertCLIFileExists(t, outputDir, "apps/api/package.json")
	assertCLIFileExists(t, outputDir, "apps/web/package.json")

	// Config should list the services
	cfg := loadCLIConfig(t, outputDir)
	assert.Len(t, cfg.Spec.Services, 2)
	assert.Equal(t, "api", cfg.Spec.Services[0].Name)
	assert.Equal(t, "web", cfg.Spec.Services[1].Name)

	// Forward ports, not container ports
	assert.Equal(t, 3010, cfg.Spec.Services[0].Port)
	assert.Equal(t, 3011, cfg.Spec.Services[1].Port)
}

func TestCLI_InitNonInteractive_WithTools(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "tools-proj")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "tools-proj",
		"--org", "acme",
		"--domain", "t.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
		"--tools", "all",
	})

	err := root.Execute()
	require.NoError(t, err)

	cfg := loadCLIConfig(t, outputDir)
	assert.Len(t, cfg.Spec.DevEnvironment.Tools, len(config.AvailableDevTools))
}

// ============================================================================
// CLI Integration: iron init validation
// ============================================================================

func TestCLI_InitNonInteractive_MissingName(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{
		"init", t.TempDir(),
		"--non-interactive",
		"--org", "acme",
		"--domain", "test.dev",
	})

	err := root.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--name is required")
}

func TestCLI_InitNonInteractive_ExistingNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "existing")
	require.NoError(t, os.MkdirAll(outputDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(outputDir, "file.txt"), []byte("x"), 0o644))

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "existing",
		"--org", "acme",
		"--domain", "e.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
	})

	err := root.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}

// ============================================================================
// CLI Integration: iron generate service
// ============================================================================

func TestCLI_GenerateService(t *testing.T) {
	// First, init a project
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "gen-svc")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "gen-svc",
		"--org", "acme",
		"--domain", "gen.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
	})
	require.NoError(t, root.Execute())

	// Change to project directory so generate can find ironplate.yaml
	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(outputDir))
	defer os.Chdir(origDir) //nolint:errcheck

	// Generate a service
	root2 := newRootCmd()
	root2.SetArgs([]string{
		"generate", "service", "auth-service",
		"--type", "node-api",
		"--group", "auth",
	})
	require.NoError(t, root2.Execute())

	// Verify service was created
	assertCLIFileExists(t, outputDir, "apps/auth-service/package.json")
	assertCLIFileExists(t, outputDir, "apps/auth-service/src/index.ts")

	// Verify Helm chart
	assertCLIFileExists(t, outputDir, "k8s/helm/gen-svc/auth/Chart.yaml")
	assertCLIFileExists(t, outputDir, "k8s/helm/gen-svc/auth/values.yaml")

	// Verify ironplate.yaml updated
	cfg := loadCLIConfig(t, outputDir)
	require.Len(t, cfg.Spec.Services, 1)
	assert.Equal(t, "auth-service", cfg.Spec.Services[0].Name)
	assert.Equal(t, "node-api", cfg.Spec.Services[0].Type)
	assert.Equal(t, "auth", cfg.Spec.Services[0].Group)
}

func TestCLI_GenerateService_DuplicateName(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "dup-svc")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "dup-svc",
		"--org", "acme",
		"--domain", "d.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
		"--example-services",
	})
	require.NoError(t, root.Execute())

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(outputDir))
	defer os.Chdir(origDir) //nolint:errcheck

	// Try to generate a service with a name that already exists
	root2 := newRootCmd()
	root2.SetArgs([]string{"generate", "service", "api", "--type", "node-api"})
	err := root2.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// ============================================================================
// CLI Integration: iron generate package
// ============================================================================

func TestCLI_GeneratePackage(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "gen-pkg")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "gen-pkg",
		"--org", "acme",
		"--domain", "g.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
	})
	require.NoError(t, root.Execute())

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(outputDir))
	defer os.Chdir(origDir) //nolint:errcheck

	root2 := newRootCmd()
	root2.SetArgs([]string{"generate", "package", "auth-utils", "--language", "node"})
	require.NoError(t, root2.Execute())

	assertCLIFileExists(t, outputDir, "packages/node/auth-utils/package.json")

	cfg := loadCLIConfig(t, outputDir)
	require.Len(t, cfg.Spec.Packages, 1)
	assert.Equal(t, "auth-utils", cfg.Spec.Packages[0].Name)
}

// ============================================================================
// CLI Integration: iron add / iron remove
// ============================================================================

func TestCLI_AddRemoveComponent(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "add-rm")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "add-rm",
		"--org", "acme",
		"--domain", "a.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
	})
	require.NoError(t, root.Execute())

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(outputDir))
	defer os.Chdir(origDir) //nolint:errcheck

	// Add kafka
	root2 := newRootCmd()
	root2.SetArgs([]string{"add", "kafka"})
	require.NoError(t, root2.Execute())

	cfg := loadCLIConfig(t, outputDir)
	assert.Contains(t, cfg.Spec.Infrastructure.Components, "kafka")
	assertCLIFileExists(t, outputDir, "k8s/helm/infra/kafka/Chart.yaml")

	// Add redis
	root3 := newRootCmd()
	root3.SetArgs([]string{"add", "redis"})
	require.NoError(t, root3.Execute())

	cfg = loadCLIConfig(t, outputDir)
	assert.Contains(t, cfg.Spec.Infrastructure.Components, "redis")

	// Remove redis
	root4 := newRootCmd()
	root4.SetArgs([]string{"remove", "redis"})
	require.NoError(t, root4.Execute())

	cfg = loadCLIConfig(t, outputDir)
	assert.NotContains(t, cfg.Spec.Infrastructure.Components, "redis")
	assert.Contains(t, cfg.Spec.Infrastructure.Components, "kafka") // kafka unaffected
}

func TestCLI_AddComponent_DependencyResolution(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "add-dep")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "add-dep",
		"--org", "acme",
		"--domain", "d.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
	})
	require.NoError(t, root.Execute())

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(outputDir))
	defer os.Chdir(origDir) //nolint:errcheck

	// Add langfuse — should auto-pull redis
	root2 := newRootCmd()
	root2.SetArgs([]string{"add", "langfuse"})
	require.NoError(t, root2.Execute())

	cfg := loadCLIConfig(t, outputDir)
	assert.Contains(t, cfg.Spec.Infrastructure.Components, "langfuse")
	assert.Contains(t, cfg.Spec.Infrastructure.Components, "redis") // auto-pulled
}

func TestCLI_RemoveComponent_WithDependents(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "rm-dep")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "rm-dep",
		"--org", "acme",
		"--domain", "r.dev",
		"--language", "node",
		"--provider", "gcp",
		"--preset", "full",
	})
	require.NoError(t, root.Execute())

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(outputDir))
	defer os.Chdir(origDir) //nolint:errcheck

	// Try to remove external-secrets (required by argocd) — should fail
	root2 := newRootCmd()
	root2.SetArgs([]string{"remove", "external-secrets"})
	err := root2.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required by")
}

// ============================================================================
// CLI Integration: iron validate
// ============================================================================

func TestCLI_Validate_ValidProject(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "valid")

	root := newRootCmd()
	root.SetArgs([]string{
		"init", outputDir,
		"--non-interactive",
		"--name", "valid",
		"--org", "acme",
		"--domain", "v.dev",
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
	})
	require.NoError(t, root.Execute())

	origDir, _ := os.Getwd()
	require.NoError(t, os.Chdir(outputDir))
	defer os.Chdir(origDir) //nolint:errcheck

	root2 := newRootCmd()
	root2.SetArgs([]string{"validate"})
	assert.NoError(t, root2.Execute())
}

// ============================================================================
// CLI Integration: iron list
// ============================================================================

func TestCLI_ListComponents(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"list", "components"})
	assert.NoError(t, root.Execute())
}

// ============================================================================
// CLI Integration: applyFlags
// ============================================================================

func TestApplyFlags_Language(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"node", []string{"node"}},
		{"go", []string{"go"}},
		{"mixed", []string{"node", "go"}},
		{"", nil}, // No change
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg := &config.ProjectConfig{}
			applyLanguage(cfg, tt.input)
			if tt.expected != nil {
				assert.Equal(t, tt.expected, cfg.Spec.Languages)
			}
		})
	}
}

func TestApplyFlags_Provider(t *testing.T) {
	t.Run("gcp", func(t *testing.T) {
		cfg := &config.ProjectConfig{}
		applyProvider(cfg, "gcp")
		assert.Equal(t, "gcp", cfg.Spec.Cloud.Provider)
	})

	t.Run("none disables gitops", func(t *testing.T) {
		cfg := &config.ProjectConfig{}
		cfg.Spec.GitOps.Enabled = true
		applyProvider(cfg, "none")
		assert.Equal(t, "none", cfg.Spec.Cloud.Provider)
		assert.False(t, cfg.Spec.GitOps.Enabled)
	})

	t.Run("empty is noop", func(t *testing.T) {
		cfg := &config.ProjectConfig{}
		cfg.Spec.Cloud.Provider = "original"
		applyProvider(cfg, "")
		assert.Equal(t, "original", cfg.Spec.Cloud.Provider)
	})
}

func TestSyncGitOpsFlag(t *testing.T) {
	t.Run("disables when no argocd", func(t *testing.T) {
		cfg := &config.ProjectConfig{}
		cfg.Spec.GitOps.Enabled = true
		cfg.Spec.Infrastructure.Components = []string{"kafka", "redis"}
		syncGitOpsFlag(cfg)
		assert.False(t, cfg.Spec.GitOps.Enabled)
	})

	t.Run("keeps enabled with argocd", func(t *testing.T) {
		cfg := &config.ProjectConfig{}
		cfg.Spec.GitOps.Enabled = true
		cfg.Spec.Infrastructure.Components = []string{"argocd"}
		syncGitOpsFlag(cfg)
		assert.True(t, cfg.Spec.GitOps.Enabled)
	})
}

// ============================================================================
// Helpers
// ============================================================================

func assertCLIFileExists(t *testing.T, root, relPath string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, relPath))
	assert.NoErrorf(t, err, "expected file to exist: %s", relPath)
}

func assertCLIFileNotExists(t *testing.T, root, relPath string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, relPath))
	assert.True(t, os.IsNotExist(err), "expected file NOT to exist: %s", relPath)
}

func loadCLIConfig(t *testing.T, projectDir string) *config.ProjectConfig {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, "ironplate.yaml"))
	require.NoError(t, err)
	var cfg config.ProjectConfig
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	return &cfg
}
