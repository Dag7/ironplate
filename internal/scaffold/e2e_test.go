//go:build integration

package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/templates"
)

// ============================================================================
// Helpers
// ============================================================================

// scaffoldProject creates a real scaffold using embedded templates.
func scaffoldProject(t *testing.T, cfg *config.ProjectConfig) string {
	t.Helper()
	outputDir := filepath.Join(t.TempDir(), cfg.Metadata.Name)
	s := NewScaffolder(cfg, outputDir, templates.FS)
	require.NoError(t, s.Scaffold())
	return outputDir
}

// assertFileExists verifies a file exists at the given path relative to root.
func assertFileExists(t *testing.T, root string, relPath string) {
	t.Helper()
	full := filepath.Join(root, relPath)
	_, err := os.Stat(full)
	assert.NoErrorf(t, err, "expected file to exist: %s", relPath)
}

// assertFileContains verifies a file contains a substring.
func assertFileContains(t *testing.T, root, relPath, substr string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoErrorf(t, err, "failed to read %s", relPath)
	assert.Containsf(t, string(data), substr, "file %s should contain %q", relPath, substr)
}

// assertFileNotContains verifies a file does NOT contain a substring.
func assertFileNotContains(t *testing.T, root, relPath, substr string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoErrorf(t, err, "failed to read %s", relPath)
	assert.NotContainsf(t, string(data), substr, "file %s should NOT contain %q", relPath, substr)
}

// assertValidYAML verifies a file parses as valid YAML.
func assertValidYAML(t *testing.T, root, relPath string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoErrorf(t, err, "failed to read %s", relPath)
	var out interface{}
	err = yaml.Unmarshal(data, &out)
	assert.NoErrorf(t, err, "file %s is not valid YAML", relPath)
}

// assertNoUnrenderedTemplateVars checks that no {{ .Something }} remains in a file.
func assertNoUnrenderedTemplateVars(t *testing.T, root, relPath string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return // File doesn't exist, skip
	}
	content := string(data)
	// Skip files that legitimately contain {{ (Helm templates, GitHub Actions)
	if strings.HasSuffix(relPath, ".tpl") || strings.HasSuffix(relPath, ".yaml") && strings.Contains(relPath, "templates/") {
		return
	}
	// Check for ironplate template variables that weren't rendered
	assert.NotRegexpf(t, `\{\{\s*\.Project\.`, content, "file %s has unrendered .Project template vars", relPath)
	assert.NotRegexpf(t, `\{\{\s*\.Computed\.`, content, "file %s has unrendered .Computed template vars", relPath)
}

// countFiles counts files recursively under a directory.
func countFiles(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	return count
}

// ============================================================================
// E2E: Minimal Node.js Project
// ============================================================================

func TestE2E_ScaffoldMinimalNode(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "mini-node",
			Organization: "testorg",
			Domain:       "mini.dev",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo: config.MonorepoSpec{
				PackageManager: "yarn",
				NodeVersion:    "22",
				BuildSystem:    "nx",
			},
			Cloud: config.CloudSpec{Provider: "none"},
		},
	}

	out := scaffoldProject(t, cfg)

	// Base files
	assertFileExists(t, out, "ironplate.yaml")
	assertFileExists(t, out, "README.md")
	assertFileContains(t, out, "README.md", "mini-node")

	// Monorepo node files
	assertFileExists(t, out, "package.json")
	assertFileContains(t, out, "package.json", "mini-node")
	assertFileExists(t, out, "tsconfig.base.json")

	// Dockerfiles
	assertFileExists(t, out, "dockerfiles/node.Dockerfile")

	// Helm library chart
	assertFileExists(t, out, "k8s/helm/_lib/service/Chart.yaml")

	// Local deployment manifests
	assertFileExists(t, out, "k8s/deployment/local/postgres.yaml")

	// Should NOT have Go-specific files
	_, err := os.Stat(filepath.Join(out, "go.work"))
	assert.True(t, os.IsNotExist(err), "go.work should not exist in node-only project")

	// Should NOT have devcontainer (not configured)
	_, err = os.Stat(filepath.Join(out, ".devcontainer"))
	assert.True(t, os.IsNotExist(err), ".devcontainer should not exist when not configured")

	// Should NOT have CI/CD (not configured)
	_, err = os.Stat(filepath.Join(out, ".github"))
	assert.True(t, os.IsNotExist(err), ".github should not exist when not configured")

	// Should NOT have Tilt files (not configured)
	_, err = os.Stat(filepath.Join(out, "Tiltfile"))
	assert.True(t, os.IsNotExist(err), "Tiltfile should not exist when tilt not configured")
	_, err = os.Stat(filepath.Join(out, "tilt"))
	assert.True(t, os.IsNotExist(err), "tilt/ should not exist when tilt not configured")

	// Verify output has real content (not just empty stubs)
	assert.Greater(t, countFiles(t, out), 5, "minimal project should have more than 5 files")
}

// ============================================================================
// E2E: Minimal Go Project
// ============================================================================

func TestE2E_ScaffoldMinimalGo(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "mini-go",
			Organization: "testorg",
			Domain:       "mini.dev",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"go"},
			Monorepo: config.MonorepoSpec{
				GoVersion: "1.24",
			},
			Cloud: config.CloudSpec{Provider: "none"},
		},
	}

	out := scaffoldProject(t, cfg)

	// Base files
	assertFileExists(t, out, "ironplate.yaml")
	assertFileExists(t, out, "README.md")

	// Go monorepo files
	assertFileExists(t, out, "go.work")
	assertFileContains(t, out, "go.work", "1.24")

	// Dockerfiles
	assertFileExists(t, out, "dockerfiles/go.Dockerfile")

	// Should NOT have Node-specific files
	_, err := os.Stat(filepath.Join(out, "package.json"))
	assert.True(t, os.IsNotExist(err), "package.json should not exist in go-only project")
}

// ============================================================================
// E2E: Full Stack with All Features
// ============================================================================

func TestE2E_ScaffoldFullStack(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "full-platform",
			Organization: "acme",
			Domain:       "platform.dev",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node", "go"},
			Monorepo: config.MonorepoSpec{
				PackageManager: "yarn",
				NodeVersion:    "22",
				GoVersion:      "1.24",
				BuildSystem:    "nx",
			},
			Cloud: config.CloudSpec{
				Provider: "gcp",
				Region:   "us-central1",
			},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "hasura", "dapr", "redis", "observability", "argocd"},
			},
			DevEnvironment: config.DevEnvSpec{
				Type:     "devcontainer",
				K8sLocal: "k3d",
				DevTool:  "tilt",
			},
			CICD: config.CICDSpec{
				Platform: "github-actions",
			},
			GitOps: config.GitOpsSpec{
				Enabled:      true,
				Tool:         "argocd",
				ImageUpdater: true,
			},
			AI: config.AISpec{
				ClaudeCode: true,
				ClaudeMD:   true,
			},
		},
	}

	out := scaffoldProject(t, cfg)

	// --- Base ---
	assertFileExists(t, out, "ironplate.yaml")
	assertFileContains(t, out, "ironplate.yaml", "full-platform")
	assertFileExists(t, out, "README.md")

	// --- Monorepo: Both languages ---
	assertFileExists(t, out, "package.json")
	assertFileExists(t, out, "go.work")

	// --- DevContainer ---
	assertFileExists(t, out, ".devcontainer/devcontainer.json")
	assertFileExists(t, out, ".devcontainer/Dockerfile")
	assertFileExists(t, out, ".devcontainer/scripts/init_host.sh")
	assertFileExists(t, out, ".devcontainer/scripts/post_cmd.sh")
	assertFileExists(t, out, ".devcontainer/scripts/copy_configs.sh")
	assertFileExists(t, out, ".devcontainer/scripts/set-docker-permissions.sh")

	// Verify .claude volume mount
	assertFileContains(t, out, ".devcontainer/devcontainer.json", "claude-config")
	// Verify init script is used instead of inline command
	assertFileContains(t, out, ".devcontainer/devcontainer.json", "init_host.sh")

	// --- k3d ---
	assertFileExists(t, out, ".devcontainer/k3s/cluster-config.yaml")
	assertFileContains(t, out, ".devcontainer/k3s/cluster-config.yaml", "full-platform")

	// --- Dockerfiles ---
	assertFileExists(t, out, "dockerfiles/node.Dockerfile")
	assertFileExists(t, out, "dockerfiles/go.Dockerfile")

	// --- Tilt ---
	assertFileExists(t, out, "Tiltfile")
	assertFileContains(t, out, "Tiltfile", "full-platform")
	assertFileContains(t, out, "Tiltfile", "registry.tilt")
	assertFileContains(t, out, "Tiltfile", "load('./utils/tilt/node.tilt'")
	assertFileContains(t, out, "Tiltfile", "load('./utils/tilt/go.utils.tilt'")
	assertFileContains(t, out, "Tiltfile", "allow_k8s_contexts('k3d-full-platform')")

	// --- Tilt registry + profiles ---
	assertFileExists(t, out, "tilt/registry.yaml")
	assertValidYAML(t, out, "tilt/registry.yaml")
	assertFileContains(t, out, "tilt/registry.yaml", "kafka:")
	assertFileContains(t, out, "tilt/registry.yaml", "hasura:")
	assertFileContains(t, out, "tilt/registry.yaml", "redis:")
	assertFileContains(t, out, "tilt/registry.yaml", "observability:")
	assertFileContains(t, out, "tilt/registry.yaml", "dapr:")

	assertFileExists(t, out, "tilt/profiles.yaml")
	assertValidYAML(t, out, "tilt/profiles.yaml")
	assertFileContains(t, out, "tilt/profiles.yaml", "active: full")
	assertFileContains(t, out, "tilt/profiles.yaml", "minimal:")
	assertFileContains(t, out, "tilt/profiles.yaml", "infra-only:")

	// --- Tilt utilities ---
	assertFileExists(t, out, "utils/tilt/registry.tilt")
	assertFileContains(t, out, "utils/tilt/registry.tilt", "resolve_profile")
	assertFileExists(t, out, "utils/tilt/node.tilt")
	assertFileExists(t, out, "utils/tilt/go.utils.tilt")
	assertFileExists(t, out, "utils/tilt/dev.utils.tilt")
	assertFileExists(t, out, "utils/tilt/packages.tilt")
	assertFileExists(t, out, "utils/tilt/dict.tilt")

	// --- Helm library chart ---
	assertFileExists(t, out, "k8s/helm/_lib/service/Chart.yaml")
	assertValidYAML(t, out, "k8s/helm/_lib/service/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/_lib/service/templates/_service.tpl")

	// --- Ingress chart ---
	assertFileExists(t, out, "k8s/helm/ingress/Chart.yaml")

	// --- Local deployment ---
	assertFileExists(t, out, "k8s/deployment/local/postgres.yaml")
	assertFileExists(t, out, "k8s/deployment/local/redis.yaml")
	assertFileExists(t, out, "k8s/deployment/local/traefik.yaml")
	assertFileExists(t, out, "k8s/deployment/local/Tiltfile")

	// --- Infrastructure components ---
	assertFileExists(t, out, "k8s/helm/infra/kafka/Chart.yaml")
	assertValidYAML(t, out, "k8s/helm/infra/kafka/values.yaml")
	assertFileContains(t, out, "k8s/helm/infra/kafka/values.yaml", "full-platform")

	assertFileExists(t, out, "k8s/helm/infra/hasura/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/infra/dapr/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/infra/redis/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/infra/observability/Chart.yaml")

	// external-secrets (auto-pulled by argocd dependency)
	assertFileExists(t, out, "k8s/helm/infra/external-secrets/Chart.yaml")

	// --- ArgoCD ---
	assertFileExists(t, out, "k8s/argocd/charts/apps/Chart.yaml")
	assertValidYAML(t, out, "k8s/argocd/charts/apps/Chart.yaml")
	assertFileContains(t, out, "k8s/argocd/charts/apps/Chart.yaml", "full-platform-apps")

	assertFileExists(t, out, "k8s/argocd/charts/apps/values.yaml")
	assertFileContains(t, out, "k8s/argocd/charts/apps/values.yaml", "full-platform")
	assertFileContains(t, out, "k8s/argocd/charts/apps/values.yaml", "gcp")

	assertFileExists(t, out, "k8s/argocd/charts/apps/values-staging.yaml")
	assertFileContains(t, out, "k8s/argocd/charts/apps/values-staging.yaml", "newest-build")

	assertFileExists(t, out, "k8s/argocd/charts/apps/values-production.yaml")
	assertFileContains(t, out, "k8s/argocd/charts/apps/values-production.yaml", "semver")

	assertFileExists(t, out, "k8s/argocd/charts/apps/templates/applications.yaml")
	assertFileExists(t, out, "k8s/argocd/charts/apps/templates/image-updaters.yaml")
	assertFileExists(t, out, "k8s/argocd/charts/apps/templates/project.yaml")
	assertFileExists(t, out, "k8s/argocd/charts/apps/templates/_helpers.tpl")

	// ArgoCD bootstrap apps
	assertFileExists(t, out, "k8s/argocd/apps/staging/apps.yaml")
	assertFileContains(t, out, "k8s/argocd/apps/staging/apps.yaml", "full-platform-apps-staging")
	assertFileExists(t, out, "k8s/argocd/apps/staging/infra.yaml")
	assertFileContains(t, out, "k8s/argocd/apps/staging/infra.yaml", "kafka-staging")
	assertFileContains(t, out, "k8s/argocd/apps/staging/infra.yaml", "hasura-staging")
	assertFileContains(t, out, "k8s/argocd/apps/staging/infra.yaml", "external-secrets-staging")
	assertFileExists(t, out, "k8s/argocd/apps/staging/ingress.yaml")

	assertFileExists(t, out, "k8s/argocd/apps/production/apps.yaml")
	assertFileExists(t, out, "k8s/argocd/apps/production/infra.yaml")
	assertFileExists(t, out, "k8s/argocd/apps/production/ingress.yaml")

	// ArgoCD scripts
	assertFileExists(t, out, "k8s/argocd/scripts/troubleshoot.sh")
	assertFileContains(t, out, "k8s/argocd/scripts/troubleshoot.sh", "full-platform")

	// --- IaC (GCP Pulumi) ---
	assertFileExists(t, out, "iac/pulumi/Pulumi.yaml")

	// --- CI/CD ---
	assertFileExists(t, out, ".github/workflows/ci.yaml")
	assertFileExists(t, out, ".github/workflows/build.yaml")
	assertFileExists(t, out, ".github/workflows/staging.yaml")
	assertFileExists(t, out, ".github/workflows/production.yaml")
	assertFileExists(t, out, ".github/workflows/release.yaml")
	assertFileExists(t, out, ".github/workflows/_deploy-argocd.yaml")
	assertFileExists(t, out, ".github/actions/build-push-image/action.yaml")
	assertFileExists(t, out, ".github/actions/gke-setup/action.yaml")
	assertFileExists(t, out, ".github/actions/hasura-migrate/action.yaml")
	assertFileExists(t, out, ".github/actions/helm-deploy/action.yaml")
	assertFileExists(t, out, ".github/actions/external-secrets/action.yaml")
	assertFileExists(t, out, ".github/scripts/detect-affected.sh")
	assertFileExists(t, out, ".github/dependabot.yml")

	// CI/CD should have rendered template vars
	assertFileContains(t, out, ".github/workflows/ci.yaml", "yarn")

	// --- Scripts ---
	assertFileExists(t, out, "scripts/utils/nodemon-runner.cjs")

	// --- AI ---
	assertFileExists(t, out, "header.md")
	assertFileContains(t, out, "header.md", "full-platform")
	assertFileExists(t, out, ".claude/skills")

	// --- Template variables fully resolved ---
	assertNoUnrenderedTemplateVars(t, out, "ironplate.yaml")
	assertNoUnrenderedTemplateVars(t, out, "k8s/argocd/charts/apps/values.yaml")
	assertNoUnrenderedTemplateVars(t, out, "k8s/argocd/apps/staging/apps.yaml")
	assertNoUnrenderedTemplateVars(t, out, "k8s/argocd/apps/staging/infra.yaml")
	assertNoUnrenderedTemplateVars(t, out, "k8s/argocd/scripts/troubleshoot.sh")
	assertNoUnrenderedTemplateVars(t, out, ".devcontainer/devcontainer.json")

	// Overall file count sanity check
	assert.Greater(t, countFiles(t, out), 50, "full project should have 50+ files")
}

// ============================================================================
// E2E: ArgoCD Sync-Wave Ordering
// ============================================================================

func TestE2E_ArgoCDSyncWaveOrdering(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "wave-test",
			Organization: "acme",
			Domain:       "wave.dev",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
			Cloud:     config.CloudSpec{Provider: "gcp", Region: "us-central1"},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "hasura", "dapr", "redis", "observability", "external-secrets", "argocd"},
			},
			GitOps: config.GitOpsSpec{Enabled: true, Tool: "argocd", ImageUpdater: true},
		},
	}

	out := scaffoldProject(t, cfg)

	// Read staging infra.yaml and verify sync-wave order
	data, err := os.ReadFile(filepath.Join(out, "k8s/argocd/apps/staging/infra.yaml"))
	require.NoError(t, err)
	content := string(data)

	// External-secrets must come first (wave -1)
	assert.Contains(t, content, "external-secrets-staging")
	extSecretsIdx := strings.Index(content, "external-secrets-staging")
	kafkaIdx := strings.Index(content, "kafka-staging")
	assert.Greater(t, kafkaIdx, extSecretsIdx, "kafka (wave 0) must come after external-secrets (wave -1)")

	// Dapr (wave 1) after kafka (wave 0)
	daprIdx := strings.Index(content, "dapr-staging")
	assert.Greater(t, daprIdx, kafkaIdx, "dapr (wave 1) must come after kafka (wave 0)")

	// Hasura (wave 2) after dapr (wave 1)
	hasuraIdx := strings.Index(content, "hasura-staging")
	assert.Greater(t, hasuraIdx, daprIdx, "hasura (wave 2) must come after dapr (wave 1)")
}

// ============================================================================
// E2E: Component Isolation — Only selected components rendered
// ============================================================================

func TestE2E_OnlySelectedComponentsRendered(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "selective",
			Organization: "acme",
			Domain:       "sel.dev",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
			Cloud:     config.CloudSpec{Provider: "none"},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "redis"},
			},
		},
	}

	out := scaffoldProject(t, cfg)

	// Selected components should exist
	assertFileExists(t, out, "k8s/helm/infra/kafka/Chart.yaml")
	assertFileExists(t, out, "k8s/helm/infra/redis/Chart.yaml")

	// Unselected components should NOT exist
	_, err := os.Stat(filepath.Join(out, "k8s/helm/infra/hasura"))
	assert.True(t, os.IsNotExist(err), "hasura should not exist when not selected")
	_, err = os.Stat(filepath.Join(out, "k8s/helm/infra/dapr"))
	assert.True(t, os.IsNotExist(err), "dapr should not exist when not selected")

	// ArgoCD should not exist
	_, err = os.Stat(filepath.Join(out, "k8s/argocd"))
	assert.True(t, os.IsNotExist(err), "argocd should not exist when not selected")
}

// ============================================================================
// E2E: Component Values contain project name
// ============================================================================

func TestE2E_ComponentValuesContainProjectName(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "my-platform",
			Organization: "acme",
			Domain:       "myplatform.dev",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
			Cloud:     config.CloudSpec{Provider: "gcp", Region: "us-central1"},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "redis", "hasura", "observability"},
			},
		},
	}

	out := scaffoldProject(t, cfg)

	// Each component's values.yaml should have projectName baked in
	components := []string{"kafka", "redis", "hasura", "observability"}
	for _, comp := range components {
		t.Run(comp, func(t *testing.T) {
			valuesPath := "k8s/helm/infra/" + comp + "/values.yaml"
			assertFileExists(t, out, valuesPath)
			assertValidYAML(t, out, valuesPath)
			assertFileContains(t, out, valuesPath, "my-platform")
		})
	}
}

// ============================================================================
// E2E: CI/CD GitHub Actions escape pattern
// ============================================================================

func TestE2E_CICDGitHubActionsEscapePattern(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata: config.Metadata{
			Name:         "cicd-test",
			Organization: "acme",
			Domain:       "ci.dev",
		},
		Spec: config.ProjectSpec{
			Languages: []string{"node", "go"},
			Monorepo: config.MonorepoSpec{
				PackageManager: "yarn",
				NodeVersion:    "22",
				GoVersion:      "1.24",
			},
			Cloud: config.CloudSpec{Provider: "gcp", Region: "us-central1"},
			CICD:  config.CICDSpec{Platform: "github-actions"},
		},
	}

	out := scaffoldProject(t, cfg)

	// CI workflow should have rendered ironplate vars (package manager, versions)
	ciPath := ".github/workflows/ci.yaml"
	assertFileExists(t, out, ciPath)
	assertFileContains(t, out, ciPath, "yarn")  // package manager rendered
	assertFileContains(t, out, ciPath, `"22"`)   // node version rendered
	assertFileContains(t, out, ciPath, `"1.24"`) // go version rendered

	// Build workflow should have GitHub Actions expressions preserved
	buildPath := ".github/workflows/build.yaml"
	assertFileExists(t, out, buildPath)
	buildData, err := os.ReadFile(filepath.Join(out, buildPath))
	require.NoError(t, err)
	buildContent := string(buildData)

	// Should contain proper GitHub Actions syntax: ${{ ... }}
	assert.Regexp(t, `\$\{\{`, buildContent, "build workflow should contain GitHub Actions ${{ expressions")

	// Should NOT contain ironplate escape pattern in output
	assert.NotContains(t, buildContent, `{{ "{{" }}`, "output should not contain ironplate escape syntax")
}

// ============================================================================
// E2E: GCP-conditional files only render for GCP
// ============================================================================

func TestE2E_GCPConditionalFiles(t *testing.T) {
	t.Run("gcp", func(t *testing.T) {
		cfg := &config.ProjectConfig{
			APIVersion: "ironplate.dev/v1",
			Kind:       "Project",
			Metadata:   config.Metadata{Name: "gcp-proj", Organization: "acme", Domain: "gcp.dev"},
			Spec: config.ProjectSpec{
				Languages: []string{"node"},
				Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
				Cloud:     config.CloudSpec{Provider: "gcp", Region: "us-central1"},
				CICD:      config.CICDSpec{Platform: "github-actions"},
			},
		}

		out := scaffoldProject(t, cfg)

		// GCP-specific action should exist and have content
		gkePath := ".github/actions/gke-setup/action.yaml"
		assertFileExists(t, out, gkePath)
		data, err := os.ReadFile(filepath.Join(out, gkePath))
		require.NoError(t, err)
		assert.Greater(t, len(data), 50, "GKE setup action should have substantial content for GCP")

		// IaC should exist
		assertFileExists(t, out, "iac/pulumi/Pulumi.yaml")
	})

	t.Run("non-gcp", func(t *testing.T) {
		cfg := &config.ProjectConfig{
			APIVersion: "ironplate.dev/v1",
			Kind:       "Project",
			Metadata:   config.Metadata{Name: "aws-proj", Organization: "acme", Domain: "aws.dev"},
			Spec: config.ProjectSpec{
				Languages: []string{"node"},
				Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
				Cloud:     config.CloudSpec{Provider: "aws", Region: "us-east-1"},
				CICD:      config.CICDSpec{Platform: "github-actions"},
			},
		}

		out := scaffoldProject(t, cfg)

		// GKE-specific action should either not exist or be empty
		gkePath := filepath.Join(out, ".github/actions/gke-setup/action.yaml")
		info, err := os.Stat(gkePath)
		if err == nil {
			// If file exists, it should be empty (conditional {{ if gcp }} rendered nothing)
			assert.LessOrEqual(t, info.Size(), int64(10),
				"GKE setup action should be empty or not exist for non-GCP provider")
		}

		// GCP IaC should not exist
		_, err = os.Stat(filepath.Join(out, "iac/pulumi"))
		assert.True(t, os.IsNotExist(err), "GCP IaC should not exist for AWS provider")
	})
}

// ============================================================================
// E2E: Dependency auto-resolution
// ============================================================================

func TestE2E_DependencyAutoResolution(t *testing.T) {
	// Requesting langfuse should auto-pull redis (hard dependency)
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: "dep-test", Organization: "acme", Domain: "dep.dev"},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
			Cloud:     config.CloudSpec{Provider: "none"},
			Infrastructure: config.InfraSpec{
				Components: []string{"langfuse"},
			},
		},
	}

	out := scaffoldProject(t, cfg)

	// Langfuse should exist
	assertFileExists(t, out, "k8s/helm/infra/langfuse/Chart.yaml")

	// Redis should be auto-pulled as a dependency
	assertFileExists(t, out, "k8s/helm/infra/redis/Chart.yaml")
}

// ============================================================================
// E2E: Hasura-conditional infra.yaml entries
// ============================================================================

func TestE2E_InfraYAMLConditionalComponents(t *testing.T) {
	t.Run("with-hasura", func(t *testing.T) {
		cfg := &config.ProjectConfig{
			APIVersion: "ironplate.dev/v1",
			Kind:       "Project",
			Metadata:   config.Metadata{Name: "hasura-proj", Organization: "acme", Domain: "h.dev"},
			Spec: config.ProjectSpec{
				Languages: []string{"node"},
				Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
				Cloud:     config.CloudSpec{Provider: "gcp", Region: "us-central1"},
				Infrastructure: config.InfraSpec{
					Components: []string{"hasura", "argocd"},
				},
				GitOps: config.GitOpsSpec{Enabled: true, Tool: "argocd", ImageUpdater: true},
			},
		}

		out := scaffoldProject(t, cfg)

		data, err := os.ReadFile(filepath.Join(out, "k8s/argocd/apps/staging/infra.yaml"))
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "hasura-staging")
		assert.Contains(t, content, "external-secrets-staging")
		// Kafka should NOT be in infra since it wasn't requested
		assert.NotContains(t, content, "kafka-staging")
	})

	t.Run("without-hasura", func(t *testing.T) {
		cfg := &config.ProjectConfig{
			APIVersion: "ironplate.dev/v1",
			Kind:       "Project",
			Metadata:   config.Metadata{Name: "kafka-only", Organization: "acme", Domain: "n.dev"},
			Spec: config.ProjectSpec{
				Languages: []string{"node"},
				Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
				Cloud:     config.CloudSpec{Provider: "gcp", Region: "us-central1"},
				Infrastructure: config.InfraSpec{
					Components: []string{"kafka", "argocd"},
				},
				GitOps: config.GitOpsSpec{Enabled: true, Tool: "argocd", ImageUpdater: true},
			},
		}

		out := scaffoldProject(t, cfg)

		data, err := os.ReadFile(filepath.Join(out, "k8s/argocd/apps/staging/infra.yaml"))
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "kafka-staging")
		assert.NotContains(t, content, "hasura-staging")
	})
}

// ============================================================================
// E2E: Helm chart YAML validity
// ============================================================================

func TestE2E_HelmChartYAMLValidity(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: "yaml-test", Organization: "acme", Domain: "y.dev"},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
			Cloud:     config.CloudSpec{Provider: "gcp", Region: "us-central1"},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "redis", "hasura", "dapr", "observability", "external-secrets"},
			},
		},
	}

	out := scaffoldProject(t, cfg)

	// Validate that all scaffolded values.yaml files are valid YAML
	components := []string{"kafka", "redis", "hasura", "dapr", "observability", "external-secrets"}
	for _, comp := range components {
		t.Run(comp+"/Chart.yaml", func(t *testing.T) {
			assertValidYAML(t, out, "k8s/helm/infra/"+comp+"/Chart.yaml")
		})
		t.Run(comp+"/values.yaml", func(t *testing.T) {
			assertValidYAML(t, out, "k8s/helm/infra/"+comp+"/values.yaml")
		})
	}

	// Helm library chart
	assertValidYAML(t, out, "k8s/helm/_lib/service/Chart.yaml")
	assertValidYAML(t, out, "k8s/helm/_lib/service/values.yaml")
}

// ============================================================================
// E2E: No .tmpl extension in output
// ============================================================================

func TestE2E_NoTmplExtensionInOutput(t *testing.T) {
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: "ext-test", Organization: "acme", Domain: "e.dev"},
		Spec: config.ProjectSpec{
			Languages: []string{"node", "go"},
			Monorepo: config.MonorepoSpec{
				PackageManager: "yarn",
				NodeVersion:    "22",
				GoVersion:      "1.24",
			},
			Cloud: config.CloudSpec{Provider: "gcp", Region: "us-central1"},
			Infrastructure: config.InfraSpec{
				Components: []string{"kafka", "argocd"},
			},
			DevEnvironment: config.DevEnvSpec{Type: "devcontainer", K8sLocal: "k3d", DevTool: "tilt"},
			CICD:           config.CICDSpec{Platform: "github-actions"},
			GitOps:         config.GitOpsSpec{Enabled: true, Tool: "argocd", ImageUpdater: true},
			AI:             config.AISpec{ClaudeCode: true, ClaudeMD: true},
		},
	}

	out := scaffoldProject(t, cfg)

	// Walk all files and ensure none end in .tmpl
	err := filepath.Walk(out, func(path string, info os.FileInfo, err error) error {
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

// ============================================================================
// E2E: Empty render produces no file
// ============================================================================

func TestE2E_ConditionalFileOmission(t *testing.T) {
	// Non-GCP project with github-actions CI/CD
	cfg := &config.ProjectConfig{
		APIVersion: "ironplate.dev/v1",
		Kind:       "Project",
		Metadata:   config.Metadata{Name: "omit-test", Organization: "acme", Domain: "o.dev"},
		Spec: config.ProjectSpec{
			Languages: []string{"node"},
			Monorepo:  config.MonorepoSpec{PackageManager: "yarn", NodeVersion: "22"},
			Cloud:     config.CloudSpec{Provider: "aws", Region: "us-east-1"},
			CICD:      config.CICDSpec{Platform: "github-actions"},
			Infrastructure: config.InfraSpec{
				Components: []string{}, // No hasura, no external-secrets
			},
		},
	}

	out := scaffoldProject(t, cfg)

	// Hasura-migrate action should either not exist or be empty
	hmPath := filepath.Join(out, ".github/actions/hasura-migrate/action.yaml")
	info, err := os.Stat(hmPath)
	if err == nil {
		assert.LessOrEqual(t, info.Size(), int64(10),
			"hasura-migrate action should be empty when hasura is not selected")
	}

	// External-secrets action should either not exist or be empty
	esPath := filepath.Join(out, ".github/actions/external-secrets/action.yaml")
	info, err = os.Stat(esPath)
	if err == nil {
		assert.LessOrEqual(t, info.Size(), int64(10),
			"external-secrets action should be empty when external-secrets is not selected")
	}
}
