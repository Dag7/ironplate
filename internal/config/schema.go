// Package config defines the ironplate.yaml configuration schema and types.
package config

// ProjectConfig is the root configuration for an ironplate project.
type ProjectConfig struct {
	APIVersion string      `yaml:"apiVersion"` // "ironplate.dev/v1"
	Kind       string      `yaml:"kind"`       // "Project"
	Metadata   Metadata    `yaml:"metadata"`
	Spec       ProjectSpec `yaml:"spec"`
}

// Metadata contains project identification.
type Metadata struct {
	Name         string `yaml:"name"`         // Project name (kebab-case)
	Organization string `yaml:"organization"` // Organization/namespace
	Domain       string `yaml:"domain"`       // Base domain (e.g., myplatform.dev)
	Description  string `yaml:"description"`  // Human-readable description
}

// ProjectSpec is the main specification for the project.
type ProjectSpec struct {
	Languages      []string          `yaml:"languages"`      // ["node", "go"]
	Monorepo       MonorepoSpec      `yaml:"monorepo"`
	Cloud          CloudSpec         `yaml:"cloud"`
	Infrastructure InfraSpec         `yaml:"infrastructure"`
	DevEnvironment DevEnvSpec        `yaml:"devEnvironment"`
	CICD           CICDSpec          `yaml:"cicd"`
	GitOps         GitOpsSpec        `yaml:"gitops"`
	Observability  ObservabilitySpec `yaml:"observability"`
	AI             AISpec            `yaml:"ai"`
	Services       []ServiceSpec     `yaml:"services"`
	Packages       []PackageSpec     `yaml:"packages"`
}

// MonorepoSpec defines the monorepo tooling.
type MonorepoSpec struct {
	PackageManager string   `yaml:"packageManager"` // "yarn" | "pnpm"
	NodeVersion    string   `yaml:"nodeVersion"`    // e.g., "22"
	GoVersion      string   `yaml:"goVersion"`      // e.g., "1.24"
	BuildSystem    string   `yaml:"buildSystem"`    // "nx" | "turborepo"
	Scopes         []string `yaml:"scopes"`         // e.g., ["@myproject", "@oss"]
}

// CloudSpec defines cloud provider configuration.
type CloudSpec struct {
	Provider     string            `yaml:"provider"` // "gcp" | "aws" | "azure" | "none"
	Region       string            `yaml:"region"`
	Environments []EnvironmentSpec `yaml:"environments"`
}

// EnvironmentSpec defines a deployment environment.
type EnvironmentSpec struct {
	Name      string `yaml:"name"`      // e.g., "staging"
	ShortName string `yaml:"shortName"` // e.g., "stg"
	Project   string `yaml:"project"`   // Cloud project ID
}

// InfraSpec defines infrastructure components.
type InfraSpec struct {
	Components        []string     `yaml:"components"`        // e.g., ["kafka", "hasura", "dapr", "redis"]
	Database          DatabaseSpec `yaml:"database"`
	MessageBus        string       `yaml:"messageBus"`        // "kafka" | "rabbitmq" | "nats"
	IngressController string       `yaml:"ingressController"` // "traefik" | "nginx"
	SecretManager     string       `yaml:"secretManager"`     // "external-secrets" | "sealed-secrets"
}

// DatabaseSpec defines database configuration.
type DatabaseSpec struct {
	Type    string `yaml:"type"`    // "postgresql" | "mysql"
	Version string `yaml:"version"` // e.g., "16"
}

// DevEnvSpec defines the development environment.
type DevEnvSpec struct {
	Type     string   `yaml:"type"`     // "devcontainer" | "docker-compose"
	K8sLocal string   `yaml:"k8sLocal"` // "k3d" | "kind" | "minikube"
	DevTool  string   `yaml:"devTool"`  // "tilt" | "skaffold" | "devspace"
	Tools    []string `yaml:"tools"`    // Additional tools to install
}

// CICDSpec defines CI/CD configuration.
type CICDSpec struct {
	Platform string       `yaml:"platform"` // "github-actions" | "gitlab-ci"
	Registry RegistrySpec `yaml:"registry"`
	Auth     string       `yaml:"auth"` // "workload-identity" | "service-account-key"
}

// RegistrySpec defines container registry configuration.
type RegistrySpec struct {
	Type string `yaml:"type"` // "artifact-registry" | "ecr" | "acr" | "ghcr"
	URL  string `yaml:"url"`  // e.g., "us-central1-docker.pkg.dev"
}

// GitOpsSpec defines GitOps configuration.
type GitOpsSpec struct {
	Enabled      bool   `yaml:"enabled"`
	Tool         string `yaml:"tool"`         // "argocd" | "flux"
	ImageUpdater bool   `yaml:"imageUpdater"` // Enable image updater
	SyncWaves    bool   `yaml:"syncWaves"`    // Enable sync wave ordering
}

// ObservabilitySpec defines observability configuration.
type ObservabilitySpec struct {
	Tracing bool `yaml:"tracing"` // OTEL + Tempo
	Metrics bool `yaml:"metrics"` // Prometheus + Grafana
	Logging bool `yaml:"logging"` // Structured JSON logging
}

// AISpec defines AI tooling configuration.
type AISpec struct {
	ClaudeCode bool     `yaml:"claudeCode"` // Generate .claude/ config
	ClaudeMD   bool     `yaml:"claudeMD"`   // Generate CLAUDE.md
	Skills     []string `yaml:"skills"`     // Skills to generate
}

// ServiceSpec defines a service in the project.
type ServiceSpec struct {
	Name     string   `yaml:"name"`     // Service name (kebab-case)
	Type     string   `yaml:"type"`     // "node-api" | "go-api" | "nextjs" | "go-worker"
	Group    string   `yaml:"group"`    // Helm umbrella chart group
	Port     int      `yaml:"port"`     // Application port
	Features []string `yaml:"features"` // ["hasura", "cache", "dapr", "eventbus"]
}

// PackageSpec defines a shared package.
type PackageSpec struct {
	Name     string `yaml:"name"`     // Package name
	Scope    string `yaml:"scope"`    // e.g., "@myproject"
	Language string `yaml:"language"` // "node" | "go"
}

// HasLanguage checks if the project uses a specific language.
func (s *ProjectSpec) HasLanguage(lang string) bool {
	for _, l := range s.Languages {
		if l == lang {
			return true
		}
	}
	return false
}

// HasComponent checks if a specific infrastructure component is enabled.
func (s *InfraSpec) HasComponent(name string) bool {
	for _, c := range s.Components {
		if c == name {
			return true
		}
	}
	return false
}

// HasTool checks if a specific dev tool is enabled.
func (s *DevEnvSpec) HasTool(name string) bool {
	for _, t := range s.Tools {
		if t == name {
			return true
		}
	}
	return false
}

// AvailableDevTools lists all optional dev tools with descriptions.
var AvailableDevTools = []struct {
	Name        string
	Description string
}{
	{"operator-sdk", "Kubernetes Operator SDK"},
	{"git-secret", "Git secret management"},
	{"mc", "MinIO client (object storage)"},
	{"kompose", "Docker Compose to K8s converter"},
}
