package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ValidConfig(t *testing.T) {
	yaml := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: test-project
  organization: testorg
spec:
  languages:
    - node
  monorepo:
    packageManager: yarn
    nodeVersion: "22"
    goVersion: "1.24"
    buildSystem: nx
  cloud:
    provider: gcp
    region: us-central1
`
	cfg, err := Parse([]byte(yaml))
	require.NoError(t, err)
	assert.Equal(t, "test-project", cfg.Metadata.Name)
	assert.Equal(t, "testorg", cfg.Metadata.Organization)
	assert.Equal(t, []string{"node"}, cfg.Spec.Languages)
	assert.Equal(t, "gcp", cfg.Spec.Cloud.Provider)
}

func TestParse_MissingName(t *testing.T) {
	yaml := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: ""
  organization: testorg
spec:
  languages:
    - node
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name is required")
}

func TestParse_InvalidAPIVersion(t *testing.T) {
	yaml := `
apiVersion: v999
kind: Project
metadata:
  name: test
  organization: testorg
spec:
  languages:
    - node
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported apiVersion")
}

func TestParse_InvalidLanguage(t *testing.T) {
	yaml := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: test
  organization: testorg
spec:
  languages:
    - python
`
	_, err := Parse([]byte(yaml))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported language")
}

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig("my-app", "myorg")

	assert.Equal(t, CurrentAPIVersion, cfg.APIVersion)
	assert.Equal(t, CurrentKind, cfg.Kind)
	assert.Equal(t, "my-app", cfg.Metadata.Name)
	assert.Equal(t, "myorg", cfg.Metadata.Organization)
	assert.Equal(t, []string{"node"}, cfg.Spec.Languages)
	assert.Equal(t, "yarn", cfg.Spec.Monorepo.PackageManager)
	assert.Equal(t, "gcp", cfg.Spec.Cloud.Provider)
	assert.Equal(t, 2, len(cfg.Spec.Cloud.Environments))
}

func TestHasLanguage(t *testing.T) {
	spec := &ProjectSpec{Languages: []string{"node", "go"}}

	assert.True(t, spec.HasLanguage("node"))
	assert.True(t, spec.HasLanguage("go"))
	assert.False(t, spec.HasLanguage("python"))
}

func TestHasComponent(t *testing.T) {
	infra := &InfraSpec{Components: []string{"kafka", "redis"}}

	assert.True(t, infra.HasComponent("kafka"))
	assert.True(t, infra.HasComponent("redis"))
	assert.False(t, infra.HasComponent("hasura"))
}

func TestParse_FullConfig(t *testing.T) {
	yamlData := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: full-project
  organization: fullorg
  domain: fullplatform.dev
  description: A fully configured project
spec:
  languages:
    - node
    - go
  monorepo:
    packageManager: pnpm
    nodeVersion: "20"
    goVersion: "1.23"
    buildSystem: turborepo
    scopes:
      - "@fullorg"
      - "@oss"
  cloud:
    provider: aws
    region: us-east-1
    environments:
      - name: staging
        shortName: stg
        project: full-staging
      - name: production
        shortName: prd
        project: full-production
  infrastructure:
    components:
      - kafka
      - hasura
      - dapr
      - redis
    database:
      type: postgresql
      version: "15"
    messageBus: kafka
    ingressController: nginx
    secretManager: sealed-secrets
  devEnvironment:
    type: docker-compose
    k8sLocal: kind
    devTool: skaffold
    tools:
      - helm
      - kustomize
  cicd:
    platform: github-actions
    registry:
      type: ecr
      url: 123456789.dkr.ecr.us-east-1.amazonaws.com
    auth: service-account-key
  gitops:
    enabled: true
    tool: flux
    imageUpdater: false
    syncWaves: true
  observability:
    tracing: true
    metrics: true
    logging: false
  ai:
    claudeCode: true
    claudeMD: false
    skills:
      - deploy
      - review
  services:
    - name: api-gateway
      type: node-api
      group: core
      port: 3000
      features:
        - hasura
        - cache
    - name: worker
      type: go-worker
      group: backend
      port: 8080
      features:
        - dapr
        - eventbus
  packages:
    - name: shared-utils
      scope: "@fullorg"
      language: node
    - name: go-lib
      scope: "@fullorg"
      language: go
`
	cfg, err := Parse([]byte(yamlData))
	require.NoError(t, err)

	// Metadata
	assert.Equal(t, CurrentAPIVersion, cfg.APIVersion)
	assert.Equal(t, CurrentKind, cfg.Kind)
	assert.Equal(t, "full-project", cfg.Metadata.Name)
	assert.Equal(t, "fullorg", cfg.Metadata.Organization)
	assert.Equal(t, "fullplatform.dev", cfg.Metadata.Domain)
	assert.Equal(t, "A fully configured project", cfg.Metadata.Description)

	// Languages
	assert.Equal(t, []string{"node", "go"}, cfg.Spec.Languages)

	// Monorepo
	assert.Equal(t, "pnpm", cfg.Spec.Monorepo.PackageManager)
	assert.Equal(t, "20", cfg.Spec.Monorepo.NodeVersion)
	assert.Equal(t, "1.23", cfg.Spec.Monorepo.GoVersion)
	assert.Equal(t, "turborepo", cfg.Spec.Monorepo.BuildSystem)
	assert.Equal(t, []string{"@fullorg", "@oss"}, cfg.Spec.Monorepo.Scopes)

	// Cloud
	assert.Equal(t, "aws", cfg.Spec.Cloud.Provider)
	assert.Equal(t, "us-east-1", cfg.Spec.Cloud.Region)
	require.Len(t, cfg.Spec.Cloud.Environments, 2)
	assert.Equal(t, "staging", cfg.Spec.Cloud.Environments[0].Name)
	assert.Equal(t, "stg", cfg.Spec.Cloud.Environments[0].ShortName)
	assert.Equal(t, "full-staging", cfg.Spec.Cloud.Environments[0].Project)
	assert.Equal(t, "production", cfg.Spec.Cloud.Environments[1].Name)
	assert.Equal(t, "prd", cfg.Spec.Cloud.Environments[1].ShortName)
	assert.Equal(t, "full-production", cfg.Spec.Cloud.Environments[1].Project)

	// Infrastructure
	assert.Equal(t, []string{"kafka", "hasura", "dapr", "redis"}, cfg.Spec.Infrastructure.Components)
	assert.Equal(t, "postgresql", cfg.Spec.Infrastructure.Database.Type)
	assert.Equal(t, "15", cfg.Spec.Infrastructure.Database.Version)
	assert.Equal(t, "kafka", cfg.Spec.Infrastructure.MessageBus)
	assert.Equal(t, "nginx", cfg.Spec.Infrastructure.IngressController)
	assert.Equal(t, "sealed-secrets", cfg.Spec.Infrastructure.SecretManager)

	// DevEnvironment
	assert.Equal(t, "docker-compose", cfg.Spec.DevEnvironment.Type)
	assert.Equal(t, "kind", cfg.Spec.DevEnvironment.K8sLocal)
	assert.Equal(t, "skaffold", cfg.Spec.DevEnvironment.DevTool)
	assert.Equal(t, []string{"helm", "kustomize"}, cfg.Spec.DevEnvironment.Tools)

	// CICD
	assert.Equal(t, "github-actions", cfg.Spec.CICD.Platform)
	assert.Equal(t, "ecr", cfg.Spec.CICD.Registry.Type)
	assert.Equal(t, "123456789.dkr.ecr.us-east-1.amazonaws.com", cfg.Spec.CICD.Registry.URL)
	assert.Equal(t, "service-account-key", cfg.Spec.CICD.Auth)

	// GitOps
	assert.True(t, cfg.Spec.GitOps.Enabled)
	assert.Equal(t, "flux", cfg.Spec.GitOps.Tool)
	assert.False(t, cfg.Spec.GitOps.ImageUpdater)
	assert.True(t, cfg.Spec.GitOps.SyncWaves)

	// Observability
	assert.True(t, cfg.Spec.Observability.Tracing)
	assert.True(t, cfg.Spec.Observability.Metrics)
	assert.False(t, cfg.Spec.Observability.Logging)

	// AI
	assert.True(t, cfg.Spec.AI.ClaudeCode)
	assert.False(t, cfg.Spec.AI.ClaudeMD)
	assert.Equal(t, []string{"deploy", "review"}, cfg.Spec.AI.Skills)

	// Services
	require.Len(t, cfg.Spec.Services, 2)
	assert.Equal(t, "api-gateway", cfg.Spec.Services[0].Name)
	assert.Equal(t, "node-api", cfg.Spec.Services[0].Type)
	assert.Equal(t, "core", cfg.Spec.Services[0].Group)
	assert.Equal(t, 3000, cfg.Spec.Services[0].Port)
	assert.Equal(t, []string{"hasura", "cache"}, cfg.Spec.Services[0].Features)
	assert.Equal(t, "worker", cfg.Spec.Services[1].Name)
	assert.Equal(t, "go-worker", cfg.Spec.Services[1].Type)
	assert.Equal(t, "backend", cfg.Spec.Services[1].Group)
	assert.Equal(t, 8080, cfg.Spec.Services[1].Port)
	assert.Equal(t, []string{"dapr", "eventbus"}, cfg.Spec.Services[1].Features)

	// Packages
	require.Len(t, cfg.Spec.Packages, 2)
	assert.Equal(t, "shared-utils", cfg.Spec.Packages[0].Name)
	assert.Equal(t, "@fullorg", cfg.Spec.Packages[0].Scope)
	assert.Equal(t, "node", cfg.Spec.Packages[0].Language)
	assert.Equal(t, "go-lib", cfg.Spec.Packages[1].Name)
	assert.Equal(t, "@fullorg", cfg.Spec.Packages[1].Scope)
	assert.Equal(t, "go", cfg.Spec.Packages[1].Language)
}

func TestParse_MissingOrganization(t *testing.T) {
	yamlData := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: test-project
  organization: ""
spec:
  languages:
    - node
`
	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.organization is required")
}

func TestParse_InvalidCloudProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{name: "digitalocean", provider: "digitalocean"},
		{name: "heroku", provider: "heroku"},
		{name: "random string", provider: "foobar"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yamlData := []byte(`
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: test-project
  organization: testorg
spec:
  languages:
    - node
  cloud:
    provider: ` + tc.provider + `
`)
			_, err := Parse(yamlData)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported cloud provider")
		})
	}
}

func TestParse_EmptyLanguages(t *testing.T) {
	yamlData := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: test-project
  organization: testorg
spec:
  languages: []
`
	_, err := Parse([]byte(yamlData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have at least one entry")
}

func TestParse_MultilanguageConfig(t *testing.T) {
	tests := []struct {
		name      string
		languages string
		expected  []string
	}{
		{
			name:      "node and go",
			languages: "    - node\n    - go",
			expected:  []string{"node", "go"},
		},
		{
			name:      "go and node (reversed)",
			languages: "    - go\n    - node",
			expected:  []string{"go", "node"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yamlData := []byte(`
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: multilang
  organization: testorg
spec:
  languages:
` + tc.languages + `
`)
			cfg, err := Parse(yamlData)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, cfg.Spec.Languages)
			for _, lang := range tc.expected {
				assert.True(t, cfg.Spec.HasLanguage(lang), "expected HasLanguage(%q) to be true", lang)
			}
		})
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "garbage input",
			input: "{{{{not yaml at all!!!!",
		},
		{
			name:  "bad indentation",
			input: "apiVersion: ironplate.dev/v1\n  kind: Project\n name: broken",
		},
		{
			name:  "tabs instead of spaces",
			input: "apiVersion: ironplate.dev/v1\nkind: Project\nmetadata:\n\t\tname: test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.input))
			require.Error(t, err)
		})
	}
}

func TestParse_ServicesAndPackages(t *testing.T) {
	yamlData := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: svc-test
  organization: testorg
spec:
  languages:
    - node
    - go
  services:
    - name: web-app
      type: nextjs
      group: frontend
      port: 3000
      features:
        - cache
    - name: auth-service
      type: go-api
      group: core
      port: 8081
      features:
        - dapr
        - eventbus
        - hasura
    - name: bg-worker
      type: go-worker
      group: backend
      port: 9090
      features: []
  packages:
    - name: ui-kit
      scope: "@testorg"
      language: node
    - name: proto-gen
      scope: "@testorg"
      language: go
    - name: eslint-config
      scope: "@oss"
      language: node
`
	cfg, err := Parse([]byte(yamlData))
	require.NoError(t, err)

	// Services
	require.Len(t, cfg.Spec.Services, 3)

	svcTests := []struct {
		idx      int
		name     string
		svcType  string
		group    string
		port     int
		features []string
	}{
		{0, "web-app", "nextjs", "frontend", 3000, []string{"cache"}},
		{1, "auth-service", "go-api", "core", 8081, []string{"dapr", "eventbus", "hasura"}},
		{2, "bg-worker", "go-worker", "backend", 9090, []string{}},
	}

	for _, tc := range svcTests {
		t.Run("service_"+tc.name, func(t *testing.T) {
			svc := cfg.Spec.Services[tc.idx]
			assert.Equal(t, tc.name, svc.Name)
			assert.Equal(t, tc.svcType, svc.Type)
			assert.Equal(t, tc.group, svc.Group)
			assert.Equal(t, tc.port, svc.Port)
			assert.Equal(t, tc.features, svc.Features)
		})
	}

	// Packages
	require.Len(t, cfg.Spec.Packages, 3)

	pkgTests := []struct {
		idx      int
		name     string
		scope    string
		language string
	}{
		{0, "ui-kit", "@testorg", "node"},
		{1, "proto-gen", "@testorg", "go"},
		{2, "eslint-config", "@oss", "node"},
	}

	for _, tc := range pkgTests {
		t.Run("package_"+tc.name, func(t *testing.T) {
			pkg := cfg.Spec.Packages[tc.idx]
			assert.Equal(t, tc.name, pkg.Name)
			assert.Equal(t, tc.scope, pkg.Scope)
			assert.Equal(t, tc.language, pkg.Language)
		})
	}
}

func TestParse_InfrastructureComponents(t *testing.T) {
	yamlData := `
apiVersion: ironplate.dev/v1
kind: Project
metadata:
  name: infra-test
  organization: testorg
spec:
  languages:
    - node
  infrastructure:
    components:
      - kafka
      - hasura
      - dapr
      - redis
    database:
      type: mysql
      version: "8"
    messageBus: rabbitmq
    ingressController: traefik
    secretManager: external-secrets
`
	cfg, err := Parse([]byte(yamlData))
	require.NoError(t, err)

	infra := cfg.Spec.Infrastructure

	// Components
	assert.Equal(t, []string{"kafka", "hasura", "dapr", "redis"}, infra.Components)

	componentTests := []struct {
		name     string
		expected bool
	}{
		{"kafka", true},
		{"hasura", true},
		{"dapr", true},
		{"redis", true},
		{"nats", false},
		{"mongodb", false},
	}

	for _, tc := range componentTests {
		t.Run("component_"+tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, infra.HasComponent(tc.name))
		})
	}

	// Database
	assert.Equal(t, "mysql", infra.Database.Type)
	assert.Equal(t, "8", infra.Database.Version)

	// Other infra fields
	assert.Equal(t, "rabbitmq", infra.MessageBus)
	assert.Equal(t, "traefik", infra.IngressController)
	assert.Equal(t, "external-secrets", infra.SecretManager)
}

func TestNewDefaultConfig_Fields(t *testing.T) {
	cfg := NewDefaultConfig("my-platform", "acme")

	// Top-level
	assert.Equal(t, "ironplate.dev/v1", cfg.APIVersion)
	assert.Equal(t, "Project", cfg.Kind)

	// Metadata
	assert.Equal(t, "my-platform", cfg.Metadata.Name)
	assert.Equal(t, "acme", cfg.Metadata.Organization)
	assert.Empty(t, cfg.Metadata.Domain, "domain should be empty by default")
	assert.Empty(t, cfg.Metadata.Description, "description should be empty by default")

	// Languages
	assert.Equal(t, []string{"node"}, cfg.Spec.Languages)

	// Monorepo
	mono := cfg.Spec.Monorepo
	assert.Equal(t, "yarn", mono.PackageManager)
	assert.Equal(t, "22", mono.NodeVersion)
	assert.Equal(t, "1.24", mono.GoVersion)
	assert.Equal(t, "nx", mono.BuildSystem)
	assert.Equal(t, []string{"@acme"}, mono.Scopes)

	// Cloud
	cloud := cfg.Spec.Cloud
	assert.Equal(t, "gcp", cloud.Provider)
	assert.Equal(t, "us-central1", cloud.Region)
	require.Len(t, cloud.Environments, 2)
	assert.Equal(t, "staging", cloud.Environments[0].Name)
	assert.Equal(t, "stg", cloud.Environments[0].ShortName)
	assert.Empty(t, cloud.Environments[0].Project, "staging project should be empty by default")
	assert.Equal(t, "production", cloud.Environments[1].Name)
	assert.Equal(t, "prd", cloud.Environments[1].ShortName)
	assert.Empty(t, cloud.Environments[1].Project, "production project should be empty by default")

	// Infrastructure
	infra := cfg.Spec.Infrastructure
	assert.Nil(t, infra.Components, "components should be nil by default")
	assert.Equal(t, "postgresql", infra.Database.Type)
	assert.Equal(t, "16", infra.Database.Version)
	assert.Equal(t, "kafka", infra.MessageBus)
	assert.Equal(t, "traefik", infra.IngressController)
	assert.Equal(t, "external-secrets", infra.SecretManager)

	// DevEnvironment
	dev := cfg.Spec.DevEnvironment
	assert.Equal(t, "devcontainer", dev.Type)
	assert.Equal(t, "k3d", dev.K8sLocal)
	assert.Equal(t, "tilt", dev.DevTool)
	assert.Nil(t, dev.Tools, "tools should be nil by default")

	// CICD
	cicd := cfg.Spec.CICD
	assert.Equal(t, "github-actions", cicd.Platform)
	assert.Equal(t, "artifact-registry", cicd.Registry.Type)
	assert.Equal(t, "us-central1-docker.pkg.dev", cicd.Registry.URL)
	assert.Equal(t, "workload-identity", cicd.Auth)

	// GitOps
	gitops := cfg.Spec.GitOps
	assert.True(t, gitops.Enabled)
	assert.Equal(t, "argocd", gitops.Tool)
	assert.True(t, gitops.ImageUpdater)
	assert.True(t, gitops.SyncWaves)

	// Observability
	obs := cfg.Spec.Observability
	assert.True(t, obs.Tracing)
	assert.True(t, obs.Metrics)
	assert.True(t, obs.Logging)

	// AI
	ai := cfg.Spec.AI
	assert.True(t, ai.ClaudeCode)
	assert.True(t, ai.ClaudeMD)
	assert.Nil(t, ai.Skills, "skills should be nil by default")

	// Services and Packages should be nil/empty
	assert.Nil(t, cfg.Spec.Services, "services should be nil by default")
	assert.Nil(t, cfg.Spec.Packages, "packages should be nil by default")
}

func TestFindConfigFile(t *testing.T) {
	// Create a temp directory structure:
	//   tmpRoot/
	//     ironplate.yaml
	//     sub/
	//       deep/
	tests := []struct {
		name     string
		startDir string // relative to tmpRoot
	}{
		{name: "same directory", startDir: ""},
		{name: "one level deep", startDir: "sub"},
		{name: "two levels deep", startDir: "sub/deep"},
	}

	tmpRoot := t.TempDir()

	// Create ironplate.yaml at root
	configPath := filepath.Join(tmpRoot, DefaultConfigFile)
	err := os.WriteFile(configPath, []byte("# config"), 0644)
	require.NoError(t, err)

	// Create subdirectories
	err = os.MkdirAll(filepath.Join(tmpRoot, "sub", "deep"), 0755)
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			startDir := filepath.Join(tmpRoot, tc.startDir)
			found, err := FindConfigFile(startDir)
			require.NoError(t, err)
			assert.Equal(t, configPath, found)
		})
	}
}

func TestFindConfigFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindConfigFile(tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no ironplate.yaml found")
}

func TestPresets(t *testing.T) {
	// Verify all expected preset keys exist
	expectedKeys := []string{"minimal", "standard", "full"}
	for _, key := range expectedKeys {
		t.Run("key_exists_"+key, func(t *testing.T) {
			_, ok := Presets[key]
			assert.True(t, ok, "expected Presets to contain key %q", key)
		})
	}

	// Verify no unexpected keys
	assert.Len(t, Presets, len(expectedKeys), "Presets should have exactly %d keys", len(expectedKeys))

	// Verify preset values
	presetTests := []struct {
		name     string
		expected []string
	}{
		{
			name:     "minimal",
			expected: []string{},
		},
		{
			name:     "standard",
			expected: []string{"redis", "kafka", "hasura", "external-secrets"},
		},
		{
			name:     "full",
			expected: []string{"redis", "kafka", "hasura", "dapr", "observability", "external-secrets", "argocd", "langfuse"},
		},
	}

	for _, tc := range presetTests {
		t.Run("values_"+tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Presets[tc.name])
		})
	}
}
