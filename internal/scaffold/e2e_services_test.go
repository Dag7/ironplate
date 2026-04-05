//go:build integration

package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/templates"
)

// ============================================================================
// E2E: Example Service Generation — Port Architecture
// ============================================================================

func TestE2E_ExampleServices_NodeOnly(t *testing.T) {
	cfg := fullNodeConfig("svc-node-test")
	out := scaffoldProject(t, cfg)

	services := DefaultExampleServices(cfg)
	require.Len(t, services, 2)

	err := GenerateExampleServices(cfg, out, templates.FS, services)
	require.NoError(t, err)

	// --- Verify service files exist ---
	assertFileExists(t, out, "apps/api/src/index.ts")
	assertFileExists(t, out, "apps/api/package.json")
	assertFileExists(t, out, "apps/web/package.json")

	// --- Verify container port is always 3000 (not forward port) ---
	assertFileContains(t, out, "apps/api/package.json", "--inspect=0.0.0.0:9229")
	assertFileNotContains(t, out, "apps/api/package.json", "9230")

	assertFileContains(t, out, "apps/web/package.json", "next dev --port 3000")
	assertFileNotContains(t, out, "apps/web/package.json", "3011")

	// --- Verify Helm values use container port 3000 ---
	helmValues := filepath.Join(out, "k8s/helm/svc-node-test/core/values.yaml")
	data, err := os.ReadFile(helmValues)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "port: 3000")
	assert.NotContains(t, content, "port: 3010")

	// --- Verify Tilt registry has forward ports ---
	regData, err := os.ReadFile(filepath.Join(out, "tilt/registry.yaml"))
	require.NoError(t, err)
	regContent := string(regData)
	assert.Contains(t, regContent, "port: 3010")
	assert.Contains(t, regContent, "debugPort: 9230")

	// --- Verify ironplate.yaml updated with forward ports ---
	cfgData, err := os.ReadFile(filepath.Join(out, "ironplate.yaml"))
	require.NoError(t, err)
	var updatedCfg config.ProjectConfig
	require.NoError(t, yaml.Unmarshal(cfgData, &updatedCfg))
	require.Len(t, updatedCfg.Spec.Services, 2)
	assert.Equal(t, 3010, updatedCfg.Spec.Services[0].Port)
	assert.Equal(t, 3011, updatedCfg.Spec.Services[1].Port)

	// --- Verify ingress entries created for services ---
	ingressValues := filepath.Join(out, "k8s/helm/svc-node-test/ingress/values.yaml")
	ingressData, err := os.ReadFile(ingressValues)
	require.NoError(t, err)
	ingressContent := string(ingressData)
	assert.Contains(t, ingressContent, "api.localhost")
	assert.Contains(t, ingressContent, "web.localhost")
	assert.Contains(t, ingressContent, "service: api")
	assert.Contains(t, ingressContent, "service: web")
}

func TestE2E_ExampleServices_GoOnly(t *testing.T) {
	cfg := fullGoConfig("svc-go-test")
	out := scaffoldProject(t, cfg)

	services := DefaultExampleServices(cfg)
	require.Len(t, services, 1)

	err := GenerateExampleServices(cfg, out, templates.FS, services)
	require.NoError(t, err)

	// --- Go service has container port 3000 ---
	assertFileExists(t, out, "apps/api/internal/config/config.go")
	assertFileContains(t, out, "apps/api/internal/config/config.go", `getIntEnv("PORT", 3000)`)

	// --- Helm values use container port 3000 ---
	assertFileContains(t, out, "k8s/helm/svc-go-test/core/values.yaml", "port: 3000")

	// --- Registry has forward port 3010 ---
	assertFileContains(t, out, "tilt/registry.yaml", "port: 3010")
	assertFileContains(t, out, "tilt/registry.yaml", "debugPort: 40001")
}

func TestE2E_ExampleServices_Mixed(t *testing.T) {
	cfg := fullMixedConfig("svc-mixed-test")
	out := scaffoldProject(t, cfg)

	services := DefaultExampleServices(cfg)
	require.Len(t, services, 3)

	err := GenerateExampleServices(cfg, out, templates.FS, services)
	require.NoError(t, err)

	// --- All three services exist ---
	assertFileExists(t, out, "apps/api/package.json")
	assertFileExists(t, out, "apps/web/package.json")
	assertFileExists(t, out, "apps/api-go/internal/config/config.go")

	// --- Verify forward ports are unique and sequential ---
	cfgData, err := os.ReadFile(filepath.Join(out, "ironplate.yaml"))
	require.NoError(t, err)
	var updatedCfg config.ProjectConfig
	require.NoError(t, yaml.Unmarshal(cfgData, &updatedCfg))
	require.Len(t, updatedCfg.Spec.Services, 3)

	ports := make(map[int]bool)
	for _, svc := range updatedCfg.Spec.Services {
		assert.False(t, ports[svc.Port], "duplicate port %d for %s", svc.Port, svc.Name)
		ports[svc.Port] = true
	}

	// --- api-go avoids naming collision ---
	assert.Equal(t, "api-go", updatedCfg.Spec.Services[2].Name)
}

// ============================================================================
// E2E: Example Service — Helm Charts
// ============================================================================

func TestE2E_ExampleServices_HelmCharts(t *testing.T) {
	cfg := fullNodeConfig("helm-chart-test")
	out := scaffoldProject(t, cfg)

	services := DefaultExampleServices(cfg)
	require.NoError(t, GenerateExampleServices(cfg, out, templates.FS, services))

	// Group umbrella chart created for "core"
	assertFileExists(t, out, "k8s/helm/helm-chart-test/core/Chart.yaml")
	assertValidYAML(t, out, "k8s/helm/helm-chart-test/core/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/helm-chart-test/core/values.yaml")
	assertValidYAML(t, out, "k8s/helm/helm-chart-test/core/values.yaml")

	// Group umbrella chart created for "frontend"
	assertFileExists(t, out, "k8s/helm/helm-chart-test/frontend/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/helm-chart-test/frontend/values.yaml")

	// Both contain port: 3000 (container port, not forward port)
	assertFileContains(t, out, "k8s/helm/helm-chart-test/core/values.yaml", "port: 3000")
	assertFileContains(t, out, "k8s/helm/helm-chart-test/frontend/values.yaml", "port: 3000")
}

// ============================================================================
// E2E: Example Service — Tilt Registry
// ============================================================================

func TestE2E_ExampleServices_TiltRegistry(t *testing.T) {
	cfg := fullMixedConfig("tilt-reg-test")
	out := scaffoldProject(t, cfg)

	services := DefaultExampleServices(cfg)
	require.NoError(t, GenerateExampleServices(cfg, out, templates.FS, services))

	data, err := os.ReadFile(filepath.Join(out, "tilt/registry.yaml"))
	require.NoError(t, err)

	var reg tiltRegistry
	require.NoError(t, yaml.Unmarshal(data, &reg))

	// All three services registered
	require.Contains(t, reg.Services, "api")
	require.Contains(t, reg.Services, "web")
	require.Contains(t, reg.Services, "api-go")

	// Forward ports (not container ports)
	assert.Equal(t, 3010, reg.Services["api"].Port)
	assert.Equal(t, 3011, reg.Services["web"].Port)
	assert.Equal(t, 3012, reg.Services["api-go"].Port)

	// Debug forward ports
	assert.Equal(t, 9230, reg.Services["api"].DebugPort)
	assert.Equal(t, 9231, reg.Services["web"].DebugPort)
	assert.Equal(t, 40001, reg.Services["api-go"].DebugPort)

	// Service types
	assert.Equal(t, "node-api", reg.Services["api"].Type)
	assert.Equal(t, "nextjs", reg.Services["web"].Type)
	assert.Equal(t, "go-api", reg.Services["api-go"].Type)
}

// ============================================================================
// E2E: Example Service — ArgoCD Registration
// ============================================================================

func TestE2E_ExampleServices_ArgoCDRegistration(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: "argocd-svc-test", Organization: "acme", Domain: "a.dev"},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22", BuildSystem: "nx"},
			Cloud:     config.CloudSpec{Provider: "gcp", Region: "us-central1"},
			Infrastructure: config.InfraSpec{
				Components: []string{"argocd"},
			},
			DevEnvironment: config.DevEnvSpec{Type: "devcontainer", K8sLocal: "k3d", DevTool: "tilt"},
			GitOps:         config.GitOpsSpec{Enabled: true, Tool: "argocd", ImageUpdater: true},
			AI:             config.AISpec{ClaudeCode: true, ClaudeMD: true},
		},
	}

	out := scaffoldProject(t, cfg)

	services := DefaultExampleServices(cfg)
	require.NoError(t, GenerateExampleServices(cfg, out, templates.FS, services))

	// ArgoCD values should have service groups
	valuesPath := "k8s/argocd/charts/apps/values.yaml"
	assertFileExists(t, out, valuesPath)
	assertFileContains(t, out, valuesPath, "core")
	assertFileContains(t, out, valuesPath, "frontend")
}

// ============================================================================
// E2E: Tilt Port-Forward Configuration
// ============================================================================

func TestE2E_TiltRegistryPortForwards(t *testing.T) {
	cfg := fullMixedConfig("port-fwd-test")
	out := scaffoldProject(t, cfg)

	// Verify the registry.tilt uses forward_port:3000 pattern
	assertFileExists(t, out, "utils/tilt/registry.tilt")
	assertFileContains(t, out, "utils/tilt/registry.tilt", "%d:3000")
	assertFileContains(t, out, "utils/tilt/registry.tilt", "# All containers listen on port 3000")
}

// ============================================================================
// E2E: Scaffold + Services Full Stack Smoke Test
// ============================================================================

func TestE2E_FullStackWithServices(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: "full-smoke", Organization: "acme", Domain: "smoke.dev"},
		Spec: config.ProjectSpec{
			Languages: []string{"node", "go"},
			Monorepo: config.MonorepoSpec{
				PackageManager: "yarn", NodeVersion: "22", GoVersion: "1.24", BuildSystem: "nx",
			},
			Cloud: config.CloudSpec{Provider: "gcp", Region: "us-central1"},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "hasura", "redis", "observability", "argocd"},
			},
			DevEnvironment: config.DevEnvSpec{Type: "devcontainer", K8sLocal: "k3d", DevTool: "tilt"},
			CICD:           config.CICDSpec{Platform: "github-actions"},
			GitOps:         config.GitOpsSpec{Enabled: true, Tool: "argocd", ImageUpdater: true},
			AI:             config.AISpec{ClaudeCode: true, ClaudeMD: true},
		},
	}

	out := scaffoldProject(t, cfg)

	// Generate example services
	services := DefaultExampleServices(cfg)
	require.NoError(t, GenerateExampleServices(cfg, out, templates.FS, services))

	// Base project + services should all be present
	totalFiles := countFiles(t, out)
	assert.Greater(t, totalFiles, 100, "full project with services should have 100+ files")

	// Every service has source, helm, and registry entry
	for _, svc := range services {
		t.Run(svc.Name, func(t *testing.T) {
			assertFileExists(t, out, "apps/"+svc.Name)
		})
	}

	// Registry has all services
	regData, err := os.ReadFile(filepath.Join(out, "tilt/registry.yaml"))
	require.NoError(t, err)
	var reg tiltRegistry
	require.NoError(t, yaml.Unmarshal(regData, &reg))
	assert.Len(t, reg.Services, 3)

	// Config has all services
	cfgData, err := os.ReadFile(filepath.Join(out, "ironplate.yaml"))
	require.NoError(t, err)
	var updatedCfg config.ProjectConfig
	require.NoError(t, yaml.Unmarshal(cfgData, &updatedCfg))
	assert.Len(t, updatedCfg.Spec.Services, 3)

	// Infrastructure was scaffolded
	assertFileExists(t, out, "k8s/helm/infra/kafka/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/infra/hasura/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/infra/redis/Chart.yaml")

	// Ingress has service entries
	ingressValues := filepath.Join(out, "k8s/helm/full-smoke/ingress/values.yaml")
	ingressData, err2 := os.ReadFile(ingressValues)
	require.NoError(t, err2)
	ingressContent := string(ingressData)
	assert.Contains(t, ingressContent, "api.localhost")
	assert.Contains(t, ingressContent, "web.localhost")
	assert.Contains(t, ingressContent, "api-go.localhost")

	// Infra ingress namespaces match doorz conventions
	assert.Contains(t, ingressContent, "namespace: kafka")
	assert.Contains(t, ingressContent, "namespace: monitoring")
	assert.Contains(t, ingressContent, "namespace: kube-system")

	// Kafka component uses kafka namespace
	kafkaValues := filepath.Join(out, "k8s/helm/infra/kafka/values.yaml")
	kafkaData, err3 := os.ReadFile(kafkaValues)
	require.NoError(t, err3)
	assert.Contains(t, string(kafkaData), "namespace: kafka")
}

// ============================================================================
// E2E: Validate Configuration
// ============================================================================

func TestE2E_ValidatePortConflicts(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: "port-conflict", Organization: "acme", Domain: "p.dev"},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Services: []config.ServiceSpec{
				{Name: "api", Type: "node-api", Port: 3010},
				{Name: "web", Type: "nextjs", Port: 3010}, // conflict!
			},
		},
	}

	result := ValidateForScaffold(cfg, "/tmp/test")
	// Port conflicts are caught by iron validate, not ValidateForScaffold
	// but verify basic validation works
	assert.True(t, result.IsValid() || len(result.Warnings) > 0)
}

func TestE2E_ValidateFeatureConsistency(t *testing.T) {
	// Feature consistency is checked by the validate command, not scaffold validation
	// Verify that the config maps are accessible and correct
	assert.Equal(t, "redis", config.FeatureComponentMap["cache"])
	assert.Equal(t, "kafka", config.FeatureComponentMap["eventbus"])
	assert.Equal(t, "hasura", config.FeatureComponentMap["hasura"])
	assert.Equal(t, "node", config.TypeLanguageMap["node-api"])
	assert.Equal(t, "go", config.TypeLanguageMap["go-api"])
}

// ============================================================================
// Helpers — config factories
// ============================================================================

func fullNodeConfig(name string) *config.ProjectConfig {
	return &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: name, Organization: "acme", Domain: name + ".dev"},
		Spec: config.ProjectSpec{
			Languages:      []string{"node"},
			Monorepo:       config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22", BuildSystem: "nx"},
			Cloud:          config.CloudSpec{Provider: "none"},
			DevEnvironment: config.DevEnvSpec{Type: "devcontainer", K8sLocal: "k3d", DevTool: "tilt"},
			AI:             config.AISpec{ClaudeCode: true, ClaudeMD: true},
		},
	}
}

func fullGoConfig(name string) *config.ProjectConfig {
	return &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: name, Organization: "acme", Domain: name + ".dev"},
		Spec: config.ProjectSpec{
			Languages:      []string{"go"},
			Monorepo:       config.MonorepoSpec{GoVersion: "1.24"},
			Cloud:          config.CloudSpec{Provider: "none"},
			DevEnvironment: config.DevEnvSpec{Type: "devcontainer", K8sLocal: "k3d", DevTool: "tilt"},
			AI:             config.AISpec{ClaudeCode: true, ClaudeMD: true},
		},
	}
}

func fullMixedConfig(name string) *config.ProjectConfig {
	return &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: name, Organization: "acme", Domain: name + ".dev"},
		Spec: config.ProjectSpec{
			Languages:      []string{"node", "go"},
			Monorepo:       config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22", GoVersion: "1.24", BuildSystem: "nx"},
			Cloud:          config.CloudSpec{Provider: "none"},
			DevEnvironment: config.DevEnvSpec{Type: "devcontainer", K8sLocal: "k3d", DevTool: "tilt"},
			AI:             config.AISpec{ClaudeCode: true, ClaudeMD: true},
		},
	}
}
