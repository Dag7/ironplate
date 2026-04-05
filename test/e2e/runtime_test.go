//go:build e2e

// Package e2e contains heavyweight end-to-end tests that validate the full
// ironplate pipeline: scaffold → devcontainer → k3d → tilt → service health.
//
// Prerequisites:
//   - Docker daemon running
//   - Node.js (for @devcontainers/cli via npx)
//   - ~30 min runtime (devcontainer build, k3d cluster, tilt deploy)
//
// Run: make test-e2e
//
// These tests are intentionally separated from unit/integration tests because
// they require real infrastructure and significant time to complete.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Configuration
// =============================================================================

const (
	projectName = "e2e-irontest"
	projectOrg  = "testorg"
	projectDom  = "e2e-test.dev"

	// Timeouts for each phase
	buildTimeout          = 2 * time.Minute
	scaffoldTimeout       = 1 * time.Minute
	devcontainerUpTimeout = 15 * time.Minute
	k3dInitTimeout        = 5 * time.Minute
	tiltCITimeout         = 15 * time.Minute
	healthCheckTimeout    = 2 * time.Minute
)

// =============================================================================
// Test state shared across subtests (sequential execution)
// =============================================================================

type e2eState struct {
	ironBinary    string // path to compiled iron binary
	projectDir    string // scaffolded project directory
	workspaceID   string // devcontainer workspace ID for cleanup
	containerID   string // docker container ID
	clusterReady  bool   // whether k3d cluster initialized successfully
	tiltStarted   bool   // whether tilt was started (for cleanup)
	registryPort  int    // host port for k3d registry (avoids conflicts)
}

// =============================================================================
// Main E2E Test
// =============================================================================

func TestE2E_FullRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping runtime E2E in short mode")
	}

	state := &e2eState{}

	// --- Prerequisites ---
	requireTool(t, "docker", "Docker must be running")
	requireTool(t, "npx", "Node.js (npx) required for @devcontainers/cli")
	requireDockerRunning(t)

	// Clean up any stale resources from previous failed E2E runs
	cleanupStaleE2EResources(t)

	// Shared temp directory that persists across all subtests
	sharedTmp := t.TempDir()

	// Cleanup: tear everything down when the test exits
	t.Cleanup(func() { cleanup(t, state) })

	// --- Phase 1: Build iron binary ---
	t.Run("01_build_iron", func(t *testing.T) {
		state.ironBinary = buildIron(t, sharedTmp)
	})
	if state.ironBinary == "" {
		t.Fatal("iron binary not built, cannot continue")
	}

	// --- Phase 2: Scaffold project ---
	t.Run("02_scaffold_project", func(t *testing.T) {
		state.projectDir = scaffoldProject(t, state.ironBinary, sharedTmp)
	})
	if state.projectDir == "" {
		t.Fatal("project not scaffolded, cannot continue")
	}

	// --- Phase 2b: Patch k3d registry port to avoid conflicts ---
	state.registryPort = findAvailablePort(t, 5005)
	if state.registryPort != 5005 {
		patchRegistryPort(t, state.projectDir, state.registryPort)
		t.Logf("Patched k3d registry port: 5005 → %d (avoiding port conflict)", state.registryPort)
	}

	// --- Phase 3: Validate generated files ---
	t.Run("03_validate_generated_files", func(t *testing.T) {
		t.Run("devcontainer_json", func(t *testing.T) {
			validateDevcontainerJSON(t, state.projectDir)
		})
		t.Run("dockerfile", func(t *testing.T) {
			validateDockerfile(t, state.projectDir)
		})
		t.Run("k3d_cluster_config", func(t *testing.T) {
			validateK3dClusterConfig(t, state.projectDir)
		})
		t.Run("k3d_registries", func(t *testing.T) {
			validateK3dRegistries(t, state.projectDir)
		})
		t.Run("k3d_init_script", func(t *testing.T) {
			validateK3dInitScript(t, state.projectDir)
		})
		t.Run("tiltfile", func(t *testing.T) {
			validateTiltfile(t, state.projectDir)
		})
		t.Run("tilt_registry", func(t *testing.T) {
			validateTiltRegistry(t, state.projectDir)
		})
		t.Run("tilt_profiles", func(t *testing.T) {
			validateTiltProfiles(t, state.projectDir)
		})
		t.Run("tilt_utilities", func(t *testing.T) {
			validateTiltUtilities(t, state.projectDir)
		})
		t.Run("helm_charts", func(t *testing.T) {
			validateHelmCharts(t, state.projectDir)
		})
		t.Run("service_source_files", func(t *testing.T) {
			validateServiceSources(t, state.projectDir)
		})
		t.Run("ironplate_yaml_consistency", func(t *testing.T) {
			validateConfigConsistency(t, state.projectDir)
		})
		t.Run("cross_layer_consistency", func(t *testing.T) {
			validateCrossLayerConsistency(t, state.projectDir)
		})
		t.Run("no_tmpl_extensions", func(t *testing.T) {
			validateNoTmplExtensions(t, state.projectDir)
		})
	})

	// --- Phase 4+: Runtime tests (devcontainer → k3d → tilt → health) ---
	// These require a running devcontainer and are gated behind state checks.
	// When running only phases 1-3 (e.g., -run '0[123]'), these are skipped cleanly.

	t.Run("04_devcontainer_up", func(t *testing.T) {
		if state.projectDir == "" {
			t.Skip("project not scaffolded")
		}
		state.containerID = devcontainerUp(t, state.projectDir)
	})

	t.Run("05_devcontainer_environment", func(t *testing.T) {
		if state.containerID == "" {
			t.Skip("devcontainer not running")
		}
		t.Run("tools_installed", func(t *testing.T) {
			validateDevcontainerTools(t, state.containerID)
		})
		t.Run("workspace_mounted", func(t *testing.T) {
			validateWorkspaceMount(t, state.containerID)
		})
	})

	t.Run("06_k3d_cluster_init", func(t *testing.T) {
		if state.containerID == "" {
			t.Skip("devcontainer not running")
		}
		initK3dCluster(t, state.containerID)
		state.clusterReady = true
	})

	t.Run("07_k3d_cluster_ready", func(t *testing.T) {
		if !state.clusterReady {
			t.Skip("k3d cluster not initialized")
		}
		t.Run("nodes_ready", func(t *testing.T) {
			verifyK3dNodes(t, state.containerID)
		})
		t.Run("namespace_exists", func(t *testing.T) {
			verifyNamespace(t, state.containerID)
		})
		t.Run("context_set", func(t *testing.T) {
			verifyK8sContext(t, state.containerID)
		})
	})

	t.Run("08_tilt_ci", func(t *testing.T) {
		if !state.clusterReady {
			t.Skip("k3d cluster not initialized")
		}
		state.tiltStarted = true
		runTiltCI(t, state.containerID)
	})

	t.Run("09_services_healthy", func(t *testing.T) {
		if !state.tiltStarted {
			t.Skip("tilt not started")
		}
		t.Run("tilt_resources_ok", func(t *testing.T) {
			verifyTiltResources(t, state.containerID)
		})
		t.Run("api_healthz", func(t *testing.T) {
			verifyServiceHealth(t, state.containerID, "api", 3010)
		})
		t.Run("web_healthz", func(t *testing.T) {
			verifyServiceHealth(t, state.containerID, "web", 3011)
		})
	})
}

// =============================================================================
// Phase 1: Build iron binary
// =============================================================================

func buildIron(t *testing.T, binDir string) string {
	t.Helper()
	repoRoot := findRepoRoot(t)
	binPath := filepath.Join(binDir, "iron")

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, "./cmd/iron")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build iron binary:\n%s", string(out))
	require.FileExists(t, binPath)

	t.Logf("Built iron binary: %s", binPath)
	return binPath
}

// =============================================================================
// Phase 2: Scaffold project
// =============================================================================

func scaffoldProject(t *testing.T, ironBin string, parentDir string) string {
	t.Helper()
	projectDir := filepath.Join(parentDir, projectName)

	ctx, cancel := context.WithTimeout(context.Background(), scaffoldTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ironBin,
		"init", projectDir,
		"--non-interactive",
		"--name", projectName,
		"--org", projectOrg,
		"--domain", projectDom,
		"--language", "node",
		"--provider", "none",
		"--preset", "minimal",
		"--example-services",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "iron init failed:\n%s", string(out))
	require.DirExists(t, projectDir)

	t.Logf("Scaffolded project: %s", projectDir)
	return projectDir
}

// =============================================================================
// Phase 3: Validate generated files
// =============================================================================

func validateDevcontainerJSON(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, ".devcontainer", "devcontainer.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "devcontainer.json missing")

	// Strip comments (devcontainer.json supports // comments which are not valid JSON)
	cleaned := stripJSONComments(data)

	var dc map[string]interface{}
	require.NoError(t, json.Unmarshal(cleaned, &dc), "devcontainer.json is not valid JSON")

	// Required top-level fields
	assert.Contains(t, dc, "name")
	assert.Contains(t, dc, "build")
	assert.Contains(t, dc, "runArgs")
	assert.Contains(t, dc, "mounts")
	assert.Contains(t, dc, "forwardPorts")
	assert.Contains(t, dc, "customizations")
	assert.Contains(t, dc, "postCreateCommand")

	// Name matches project
	assert.Equal(t, projectName, dc["name"])

	// Verify mounts contain required entries
	mounts, ok := dc["mounts"].([]interface{})
	require.True(t, ok, "mounts should be an array")
	mountStr := fmt.Sprintf("%v", mounts)
	assert.Contains(t, mountStr, "docker.sock", "should mount docker socket")
	assert.Contains(t, mountStr, ".kube", "should mount kube config")
	assert.Contains(t, mountStr, "claude-config", "should mount claude config volume")

	// Verify forward ports
	ports, ok := dc["forwardPorts"].([]interface{})
	require.True(t, ok, "forwardPorts should be an array")
	portNums := make([]int, 0, len(ports))
	for _, p := range ports {
		if n, ok := p.(float64); ok {
			portNums = append(portNums, int(n))
		}
	}
	assert.GreaterOrEqual(t, len(portNums), 3, "should forward at least 3 ports")
	assert.Contains(t, portNums, 8081, "should forward traefik HTTP port")
	assert.Contains(t, portNums, 10350, "should forward tilt UI port")

	// Verify VS Code extensions include essentials
	customizations, ok := dc["customizations"].(map[string]interface{})
	require.True(t, ok)
	vscode, ok := customizations["vscode"].(map[string]interface{})
	require.True(t, ok)
	extensions, ok := vscode["extensions"].([]interface{})
	require.True(t, ok)
	extStr := fmt.Sprintf("%v", extensions)
	assert.Contains(t, extStr, "ms-azuretools.vscode-docker")
	assert.Contains(t, extStr, "tilt-dev.tiltfile")

	t.Logf("devcontainer.json valid: %d mounts, %d ports, %d extensions",
		len(mounts), len(ports), len(extensions))
}

func validateDockerfile(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, ".devcontainer", "Dockerfile")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Dockerfile missing")
	content := string(data)

	// Must have a FROM base image
	assert.Contains(t, content, "FROM", "Dockerfile must have FROM instruction")
	assert.Contains(t, content, "mcr.microsoft.com", "should use Microsoft devcontainer base")

	// Must install core K8s tools
	assert.Contains(t, content, "kubectl", "should install kubectl")
	assert.Contains(t, content, "helm", "should install helm")
	assert.Contains(t, content, "k3d", "should install k3d")
	assert.Contains(t, content, "tilt", "should install tilt")

	// Must install iron CLI
	assert.Contains(t, content, "iron", "should install iron CLI")

	// Must set up shell completions
	assert.Contains(t, content, "completion", "should set up shell completions")

	// Project name should not appear in Dockerfile (it's a generic image)
	// But conditional sections might reference computed values
	assert.NotContains(t, content, "{{", "should not contain unrendered template vars")
}

func validateK3dClusterConfig(t *testing.T, projectDir string) {
	t.Helper()
	clusterName := projectName + "-cluster"
	path := filepath.Join(projectDir, ".devcontainer", "k3s", clusterName+".config.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "k3d cluster config missing")

	var cfg map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &cfg), "cluster config is not valid YAML")

	// API version and kind
	assert.Equal(t, "k3d.io/v1alpha5", cfg["apiVersion"])
	assert.Equal(t, "Simple", cfg["kind"])

	// Metadata
	metadata, ok := cfg["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, clusterName, metadata["name"])

	// Servers count
	assert.Equal(t, 1, cfg["servers"])

	// Registries
	registries, ok := cfg["registries"].(map[string]interface{})
	require.True(t, ok)
	create, ok := registries["create"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, fmt.Sprintf("k3d-%s-registry", projectName), create["name"])
	hostPort := fmt.Sprintf("%v", create["hostPort"])
	assert.NotEmpty(t, hostPort, "registry hostPort should be set")

	// Ports
	ports, ok := cfg["ports"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(ports), 2, "should have HTTP and HTTPS port mappings")

	// Options
	options, ok := cfg["options"].(map[string]interface{})
	require.True(t, ok)
	k3dOpts, ok := options["k3d"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, k3dOpts["wait"], "cluster should wait for readiness")

	t.Logf("k3d cluster config valid: cluster=%s, ports=%d", clusterName, len(ports))
}

func validateK3dRegistries(t *testing.T, projectDir string) {
	t.Helper()
	clusterName := projectName + "-cluster"
	path := filepath.Join(projectDir, ".devcontainer", "k3s", clusterName+".registries.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "k3d registries config missing")

	var reg map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &reg), "registries config is not valid YAML")

	// Should have mirrors
	mirrors, ok := reg["mirrors"].(map[string]interface{})
	require.True(t, ok, "should have mirrors section")
	assert.Contains(t, mirrors, "docker.io", "should mirror docker.io")

	// Should have configs with insecure_skip_verify
	configs, ok := reg["configs"].(map[string]interface{})
	require.True(t, ok, "should have configs section")
	registryKey := fmt.Sprintf("k3d-%s-registry:5000", projectName)
	assert.Contains(t, configs, registryKey)
}

func validateK3dInitScript(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, ".devcontainer", "k3s", "init-cluster.sh")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "init-cluster.sh missing")
	content := string(data)

	assert.Contains(t, content, "#!/bin/bash", "should have bash shebang")
	assert.Contains(t, content, "set -euo pipefail", "should use strict mode")
	assert.Contains(t, content, projectName+"-cluster", "should reference correct cluster name")
	assert.Contains(t, content, "k3d cluster create", "should create k3d cluster")
	assert.Contains(t, content, "kubectl apply", "should apply manifests")
	assert.Contains(t, content, fmt.Sprintf("--namespace=%s", projectName), "should set project namespace")
	assert.NotContains(t, content, "{{", "should not contain unrendered template vars")
}

func validateTiltfile(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, "Tiltfile")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Tiltfile missing")
	content := string(data)

	// K8s context restriction
	assert.Contains(t, content, fmt.Sprintf("allow_k8s_contexts('k3d-%s')", projectName),
		"Tiltfile should restrict to project's k3d context")

	// Load statements for node (since we scaffold node-only)
	assert.Contains(t, content, "node.tilt", "should load node tilt utilities")
	assert.Contains(t, content, "registry.tilt", "should load registry tilt utilities")
	assert.Contains(t, content, "dev.utils.tilt", "should load dev utilities")

	// Service builder dict
	assert.Contains(t, content, "node-api", "should have node-api builder")
	assert.Contains(t, content, "nextjs", "should have nextjs builder")

	// Docker prune settings
	assert.Contains(t, content, "docker_prune_settings", "should configure docker prune")

	// No unrendered template vars
	assert.NotContains(t, content, "{{ .", "should not contain unrendered ironplate template vars")
}

func validateTiltRegistry(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, "tilt", "registry.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "tilt/registry.yaml missing")

	var reg struct {
		Services       map[string]interface{} `yaml:"services"`
		Infrastructure map[string]interface{} `yaml:"infrastructure"`
	}
	require.NoError(t, yaml.Unmarshal(data, &reg), "registry.yaml is not valid YAML")

	// Infrastructure section should have postgres (always present in local dev)
	require.Contains(t, reg.Infrastructure, "postgres", "should have postgres in infrastructure")

	t.Logf("Tilt registry: %d services, %d infra", len(reg.Services), len(reg.Infrastructure))
}

func validateTiltProfiles(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, "tilt", "profiles.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "tilt/profiles.yaml missing")

	var profiles struct {
		Active   string                 `yaml:"active"`
		Profiles map[string]interface{} `yaml:"profiles"`
	}
	require.NoError(t, yaml.Unmarshal(data, &profiles), "profiles.yaml is not valid YAML")

	assert.Equal(t, "full", profiles.Active, "default active profile should be 'full'")
	assert.Contains(t, profiles.Profiles, "minimal", "should have minimal profile")
	assert.Contains(t, profiles.Profiles, "core", "should have core profile")
	assert.Contains(t, profiles.Profiles, "full", "should have full profile")
	assert.Contains(t, profiles.Profiles, "infra-only", "should have infra-only profile")
}

func validateTiltUtilities(t *testing.T, projectDir string) {
	t.Helper()
	utils := []string{
		"utils/tilt/registry.tilt",
		"utils/tilt/node.tilt",
		"utils/tilt/dev.utils.tilt",
		"utils/tilt/packages.tilt",
		"utils/tilt/dict.tilt",
	}
	for _, u := range utils {
		path := filepath.Join(projectDir, u)
		_, err := os.Stat(path)
		assert.NoErrorf(t, err, "tilt utility missing: %s", u)
	}
}

func validateHelmCharts(t *testing.T, projectDir string) {
	t.Helper()

	// Library chart
	libChart := filepath.Join(projectDir, "k8s", "helm", projectName, "_lib", "service", "Chart.yaml")
	data, err := os.ReadFile(libChart)
	require.NoError(t, err, "Helm library Chart.yaml missing")
	var chart map[string]interface{}
	require.NoError(t, yaml.Unmarshal(data, &chart), "library Chart.yaml is not valid YAML")
	assert.Equal(t, "v2", chart["apiVersion"])
	assert.Equal(t, "library", chart["type"], "library chart should be type: library")

	// Library values
	libValues := filepath.Join(projectDir, "k8s", "helm", projectName, "_lib", "service", "values.yaml")
	assertValidYAMLFile(t, libValues)

	// Ingress chart
	ingressChart := filepath.Join(projectDir, "k8s", "helm", projectName, "ingress", "Chart.yaml")
	assertValidYAMLFile(t, ingressChart)

	// Service group charts (created by example services)
	coreChart := filepath.Join(projectDir, "k8s", "helm", projectName, "core", "Chart.yaml")
	if _, err := os.Stat(coreChart); err == nil {
		assertValidYAMLFile(t, coreChart)

		// Core values should use container port 3000
		coreValues := filepath.Join(projectDir, "k8s", "helm", projectName, "core", "values.yaml")
		valData, err := os.ReadFile(coreValues)
		require.NoError(t, err)
		assert.Contains(t, string(valData), "port: 3000", "Helm values should use container port 3000")
		assert.NotContains(t, string(valData), "port: 3010", "Helm values should NOT use forward port")
	}

	// Local deployment manifests
	assertFileExistsAt(t, filepath.Join(projectDir, "k8s", "deployment", "local", "postgres.yaml"))
	assertFileExistsAt(t, filepath.Join(projectDir, "k8s", "deployment", "local", "Tiltfile"))
}

func validateServiceSources(t *testing.T, projectDir string) {
	t.Helper()

	// api service (node-api)
	assertFileExistsAt(t, filepath.Join(projectDir, "apps", "api", "package.json"))
	assertFileExistsAt(t, filepath.Join(projectDir, "apps", "api", "src", "index.ts"))
	assertFileExistsAt(t, filepath.Join(projectDir, "apps", "api", "src", "app.ts"))

	// Verify api package.json has correct debug port (container port 9229)
	apiPkg, err := os.ReadFile(filepath.Join(projectDir, "apps", "api", "package.json"))
	require.NoError(t, err)
	assert.Contains(t, string(apiPkg), "--inspect=0.0.0.0:9229",
		"node debug should use container port 9229, not host forward port")

	// web service (nextjs)
	assertFileExistsAt(t, filepath.Join(projectDir, "apps", "web", "package.json"))

	// Verify web listens on container port 3000
	webPkg, err := os.ReadFile(filepath.Join(projectDir, "apps", "web", "package.json"))
	require.NoError(t, err)
	assert.Contains(t, string(webPkg), "next dev --port 3000",
		"nextjs should listen on container port 3000")
}

func validateConfigConsistency(t *testing.T, projectDir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, "ironplate.yaml"))
	require.NoError(t, err)

	var cfg struct {
		Metadata struct {
			Name         string `yaml:"name"`
			Organization string `yaml:"organization"`
			Domain       string `yaml:"domain"`
		} `yaml:"metadata"`
		Spec struct {
			Languages []string `yaml:"languages"`
			Services  []struct {
				Name string `yaml:"name"`
				Type string `yaml:"type"`
				Port int    `yaml:"port"`
			} `yaml:"services"`
		} `yaml:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(data, &cfg))

	assert.Equal(t, projectName, cfg.Metadata.Name)
	assert.Equal(t, projectOrg, cfg.Metadata.Organization)
	assert.Contains(t, cfg.Spec.Languages, "node")

	// Example services registered
	require.Len(t, cfg.Spec.Services, 2, "should have 2 example services")
	assert.Equal(t, "api", cfg.Spec.Services[0].Name)
	assert.Equal(t, "web", cfg.Spec.Services[1].Name)

	// Forward ports are sequential from base (3010)
	assert.Equal(t, 3010, cfg.Spec.Services[0].Port)
	assert.Equal(t, 3011, cfg.Spec.Services[1].Port)

	// No duplicate ports
	ports := map[int]string{}
	for _, svc := range cfg.Spec.Services {
		existing, dup := ports[svc.Port]
		assert.Falsef(t, dup, "port %d used by both %s and %s", svc.Port, existing, svc.Name)
		ports[svc.Port] = svc.Name
	}
}

func validateCrossLayerConsistency(t *testing.T, projectDir string) {
	t.Helper()

	// Load ironplate.yaml services
	cfgData, err := os.ReadFile(filepath.Join(projectDir, "ironplate.yaml"))
	require.NoError(t, err)
	var cfg struct {
		Spec struct {
			Services []struct {
				Name  string `yaml:"name"`
				Type  string `yaml:"type"`
				Port  int    `yaml:"port"`
				Group string `yaml:"group"`
			} `yaml:"services"`
		} `yaml:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(cfgData, &cfg))

	// Load tilt registry
	regData, err := os.ReadFile(filepath.Join(projectDir, "tilt", "registry.yaml"))
	require.NoError(t, err)
	var reg struct {
		Services map[string]struct {
			Type  string `yaml:"type"`
			Group string `yaml:"group"`
			Port  int    `yaml:"port"`
		} `yaml:"services"`
	}
	require.NoError(t, yaml.Unmarshal(regData, &reg))

	// Every service in ironplate.yaml must appear in tilt registry with matching ports
	for _, svc := range cfg.Spec.Services {
		regEntry, exists := reg.Services[svc.Name]
		assert.Truef(t, exists, "service %q in ironplate.yaml must be in tilt registry", svc.Name)
		if exists {
			assert.Equalf(t, svc.Type, regEntry.Type,
				"service %q type mismatch: config=%s registry=%s", svc.Name, svc.Type, regEntry.Type)
			assert.Equalf(t, svc.Port, regEntry.Port,
				"service %q port mismatch: config=%d registry=%d", svc.Name, svc.Port, regEntry.Port)
		}
	}

	// Verify Helm group charts exist for each service group
	groups := map[string]bool{}
	for _, svc := range cfg.Spec.Services {
		groups[svc.Group] = true
	}
	for group := range groups {
		chartPath := filepath.Join(projectDir, "k8s", "helm", projectName, group, "Chart.yaml")
		assertFileExistsAt(t, chartPath)
	}
}

func validateNoTmplExtensions(t *testing.T, projectDir string) {
	t.Helper()
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			assert.NotRegexpf(t, `\.tmpl$`, path,
				"output file should not have .tmpl extension: %s", path)
		}
		return nil
	})
	require.NoError(t, err)
}

// =============================================================================
// Phase 4: Start devcontainer
// =============================================================================

func devcontainerUp(t *testing.T, projectDir string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), devcontainerUpTimeout)
	defer cancel()

	t.Log("Starting devcontainer (this may take several minutes on first run)...")

	cmd := exec.CommandContext(ctx, "npx", "--yes", "@devcontainers/cli", "up",
		"--workspace-folder", projectDir,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "devcontainer up failed:\nstdout: %s\nstderr: %s",
		stdout.String(), stderr.String())

	// Parse the output to get container ID
	// devcontainer up outputs JSON with containerId
	var result struct {
		Outcome     string `json:"outcome"`
		ContainerID string `json:"containerId"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err == nil && result.ContainerID != "" {
		t.Logf("Devcontainer started: %s (outcome: %s)", result.ContainerID[:12], result.Outcome)
		return result.ContainerID
	}

	// Fallback: find container by label
	findCmd := exec.Command("docker", "ps", "-q",
		"--filter", fmt.Sprintf("label=devcontainer.local_folder=%s", projectDir))
	containerID, err := findCmd.Output()
	require.NoError(t, err, "could not find devcontainer ID")
	id := strings.TrimSpace(string(containerID))
	require.NotEmpty(t, id, "devcontainer container not found")

	t.Logf("Devcontainer started: %s", id[:12])
	return id
}

// =============================================================================
// Phase 5: Validate devcontainer environment
// =============================================================================

func validateDevcontainerTools(t *testing.T, containerID string) {
	t.Helper()
	tools := []string{"docker", "kubectl", "helm", "k3d", "tilt", "iron"}
	for _, tool := range tools {
		out, err := devcontainerExec(containerID, "which", tool)
		assert.NoErrorf(t, err, "tool %q not found in devcontainer: %s", tool, out)
	}
}

func validateWorkspaceMount(t *testing.T, containerID string) {
	t.Helper()
	workDir := fmt.Sprintf("/workspaces/%s", projectName)

	// Check ironplate.yaml exists in workspace
	out, err := devcontainerExec(containerID, "test", "-f", workDir+"/ironplate.yaml")
	assert.NoErrorf(t, err, "ironplate.yaml not found in workspace: %s", out)

	// Check Tiltfile exists
	out, err = devcontainerExec(containerID, "test", "-f", workDir+"/Tiltfile")
	assert.NoErrorf(t, err, "Tiltfile not found in workspace: %s", out)
}

// =============================================================================
// Phase 6: Initialize k3d cluster
// =============================================================================

func initK3dCluster(t *testing.T, containerID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), k3dInitTimeout)
	defer cancel()

	workDir := fmt.Sprintf("/workspaces/%s", projectName)
	k3sDir := workDir + "/.devcontainer/k3s"

	t.Log("Initializing k3d cluster...")

	cmd := exec.CommandContext(ctx, "docker", "exec",
		"-w", k3sDir,
		containerID,
		"bash", "-c", "chmod +x init-cluster.sh scripts/*.sh && bash init-cluster.sh",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, err, "k3d cluster init failed:\nstdout: %s\nstderr: %s",
		stdout.String(), stderr.String())

	t.Logf("k3d cluster initialized")
}

// =============================================================================
// Phase 7: Verify k3d cluster
// =============================================================================

func verifyK3dNodes(t *testing.T, containerID string) {
	t.Helper()
	out, err := devcontainerExec(containerID,
		"kubectl", "get", "nodes", "-o", "jsonpath={.items[0].status.conditions[-1].type}")
	require.NoError(t, err, "kubectl get nodes failed: %s", out)
	assert.Equal(t, "Ready", strings.TrimSpace(out), "k3d node should be Ready")
}

func verifyNamespace(t *testing.T, containerID string) {
	t.Helper()
	out, err := devcontainerExec(containerID,
		"kubectl", "get", "namespace", projectName, "-o", "jsonpath={.metadata.name}")
	require.NoError(t, err, "namespace check failed: %s", out)
	assert.Equal(t, projectName, strings.TrimSpace(out))
}

func verifyK8sContext(t *testing.T, containerID string) {
	t.Helper()
	out, err := devcontainerExec(containerID,
		"kubectl", "config", "current-context")
	require.NoError(t, err, "context check failed: %s", out)
	assert.Contains(t, strings.TrimSpace(out), projectName,
		"k8s context should reference project name")
}

// =============================================================================
// Phase 8: Run tilt ci
// =============================================================================

func runTiltCI(t *testing.T, containerID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), tiltCITimeout)
	defer cancel()

	workDir := fmt.Sprintf("/workspaces/%s", projectName)

	t.Log("Starting tilt ci (deploying all resources, waiting for health)...")

	// tilt ci runs Tilt in CI mode:
	// - Deploys all resources
	// - Waits until all are ready (or errors)
	// - Exits 0 if all healthy, non-zero otherwise
	cmd := exec.CommandContext(ctx, "docker", "exec",
		"-w", workDir,
		containerID,
		"tilt", "ci", "--port", "0",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	// Log tilt output for debugging regardless of success/failure
	if stdout.Len() > 0 {
		t.Logf("tilt stdout (last 2000 chars):\n%s", tailString(stdout.String(), 2000))
	}
	if stderr.Len() > 0 {
		t.Logf("tilt stderr (last 2000 chars):\n%s", tailString(stderr.String(), 2000))
	}

	require.NoError(t, err, "tilt ci failed — resources did not become healthy")
	t.Log("tilt ci completed successfully — all resources healthy")
}

// =============================================================================
// Phase 9: Verify services are healthy
// =============================================================================

func verifyTiltResources(t *testing.T, containerID string) {
	t.Helper()

	// After tilt ci exits successfully, query remaining resource state
	out, err := devcontainerExec(containerID,
		"kubectl", "get", "pods", "-n", projectName,
		"-o", "jsonpath={range .items[*]}{.metadata.name} {.status.phase}{'\\n'}{end}")
	if err != nil {
		t.Logf("Warning: could not list pods (tilt ci may have cleaned up): %s", out)
		return
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			assert.Equalf(t, "Running", parts[1],
				"pod %s should be Running", parts[0])
		}
	}
	t.Logf("Verified %d pods running", len(lines))
}

func verifyServiceHealth(t *testing.T, containerID string, svcName string, port int) {
	t.Helper()

	// After tilt ci exits, port-forwards are gone. Verify health via k8s pod readiness.
	// tilt ci already confirmed all resources healthy, so we just verify the pods stayed up.

	// Find pods matching the service name (labels vary by Helm chart, use field selector on name)
	out, err := devcontainerExec(containerID,
		"kubectl", "get", "pods", "-n", projectName,
		"--no-headers",
		"-o", "custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type=='Ready')].status,PHASE:.status.phase")
	require.NoError(t, err, "kubectl get pods failed: %s", out)

	// Find the pod for this service
	lines := strings.Split(strings.TrimSpace(out), "\n")
	found := false
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && strings.Contains(fields[0], svcName) {
			found = true
			assert.Equalf(t, "True", fields[1],
				"pod %s should be Ready", fields[0])
			assert.Equalf(t, "Running", fields[2],
				"pod %s should be Running", fields[0])
			t.Logf("Service %s pod %s: Ready=%s Phase=%s", svcName, fields[0], fields[1], fields[2])
			break
		}
	}
	assert.Truef(t, found, "no pod found matching service %q in namespace %s.\nPods:\n%s",
		svcName, projectName, out)
}

// =============================================================================
// Cleanup
// =============================================================================

func cleanup(t *testing.T, state *e2eState) {
	t.Helper()
	t.Log("Cleaning up E2E resources...")

	if state.containerID != "" {
		// Stop tilt if it was started
		if state.tiltStarted {
			devcontainerExec(state.containerID, "tilt", "down") //nolint:errcheck
		}

		// Delete k3d cluster
		devcontainerExec(state.containerID, //nolint:errcheck
			"k3d", "cluster", "delete", projectName+"-cluster")

		// Stop and remove the devcontainer
		exec.Command("docker", "rm", "-f", state.containerID).Run() //nolint:errcheck
		t.Logf("Cleaned up container %s", state.containerID[:min(12, len(state.containerID))])
	}
}

// =============================================================================
// Helpers
// =============================================================================

func requireTool(t *testing.T, name, msg string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("Skipping E2E: %s (%s not found)", msg, name)
	}
}

func requireDockerRunning(t *testing.T) {
	t.Helper()
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping E2E: Docker daemon not running")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	// Walk up from current directory to find the root go.mod (the one with cmd/iron)
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "cmd", "iron", "main.go")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (cmd/iron/main.go)")
		}
		dir = parent
	}
}

func devcontainerExec(containerID string, args ...string) (string, error) {
	cmdArgs := append([]string{"exec", containerID}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func assertFileExistsAt(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.NoErrorf(t, err, "expected file to exist: %s", path)
}

func assertValidYAMLFile(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoErrorf(t, err, "failed to read %s", path)
	var out interface{}
	assert.NoErrorf(t, yaml.Unmarshal(data, &out), "file is not valid YAML: %s", path)
}

// stripJSONComments removes // single-line comments from JSON (devcontainer.json uses them).
func stripJSONComments(data []byte) []byte {
	var result []byte
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("//")) {
			continue
		}
		result = append(result, line...)
		result = append(result, '\n')
	}
	return result
}

func tailString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "...\n" + s[len(s)-maxLen:]
}

// cleanupStaleE2EResources removes leftover containers/clusters from previous failed E2E runs.
func cleanupStaleE2EResources(t *testing.T) {
	t.Helper()

	// Remove stale devcontainers matching our project name
	out, _ := exec.Command("docker", "ps", "-aq", "--filter",
		fmt.Sprintf("name=vsc-%s", projectName)).Output()
	for _, id := range strings.Fields(strings.TrimSpace(string(out))) {
		t.Logf("Removing stale devcontainer: %s", id)
		exec.Command("docker", "rm", "-f", id).Run() //nolint:errcheck
	}

	// Remove stale k3d clusters from E2E
	exec.Command("docker", "exec", "-i", projectName, //nolint:errcheck
		"k3d", "cluster", "delete", projectName+"-cluster").Run()
}

// findAvailablePort returns preferredPort if available, otherwise finds a free port nearby.
func findAvailablePort(t *testing.T, preferredPort int) int {
	t.Helper()

	// Try preferred port first
	if isPortAvailable(preferredPort) {
		return preferredPort
	}
	t.Logf("Port %d is in use, finding alternative...", preferredPort)

	// Scan nearby ports
	for port := preferredPort + 1; port < preferredPort+100; port++ {
		if isPortAvailable(port) {
			return port
		}
	}
	t.Fatalf("Could not find available port near %d", preferredPort)
	return 0
}

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// patchRegistryPort replaces the default k3d registry port (5005) with a new port
// in the cluster config and devcontainer.json to avoid conflicts with existing clusters.
func patchRegistryPort(t *testing.T, projectDir string, newPort int) {
	t.Helper()
	oldPort := "5005"
	newPortStr := fmt.Sprintf("%d", newPort)

	// Patch k3d cluster config
	clusterConfig := filepath.Join(projectDir, ".devcontainer", "k3s",
		projectName+"-cluster.config.yaml")
	patchFileContent(t, clusterConfig, oldPort, newPortStr)

	// Patch devcontainer.json forwardPorts
	dcJSON := filepath.Join(projectDir, ".devcontainer", "devcontainer.json")
	patchFileContent(t, dcJSON, oldPort, newPortStr)
}

func patchFileContent(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return // file may not exist in all configurations
	}
	updated := strings.ReplaceAll(string(data), old, replacement)
	if updated != string(data) {
		require.NoError(t, os.WriteFile(path, []byte(updated), 0o644))
	}
}
