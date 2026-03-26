package config

// NewDefaultConfig creates a ProjectConfig with sensible defaults.
func NewDefaultConfig(name, org string) *ProjectConfig {
	return &ProjectConfig{
		APIVersion: CurrentAPIVersion,
		Kind:       CurrentKind,
		Metadata: Metadata{
			Name:         name,
			Organization: org,
		},
		Spec: ProjectSpec{
			Languages: []string{"node"},
			Monorepo: MonorepoSpec{
				PackageManager: "yarn",
				NodeVersion:    "22",
				GoVersion:      "1.24",
				BuildSystem:    "nx",
				Scopes:         []string{"@" + org},
			},
			Cloud: CloudSpec{
				Provider: "gcp",
				Region:   "us-central1",
				Environments: []EnvironmentSpec{
					{Name: "staging", ShortName: "stg"},
					{Name: "production", ShortName: "prd"},
				},
			},
			Infrastructure: InfraSpec{
				Database:          DatabaseSpec{Type: "postgresql", Version: "16"},
				MessageBus:        "kafka",
				IngressController: "traefik",
				SecretManager:     "external-secrets",
			},
			DevEnvironment: DevEnvSpec{
				Type:     "devcontainer",
				K8sLocal: "k3d",
				DevTool:  "tilt",
			},
			CICD: CICDSpec{
				Platform: "github-actions",
				Registry: RegistrySpec{
					Type: "artifact-registry",
					URL:  "us-central1-docker.pkg.dev",
				},
				Auth: "workload-identity",
			},
			GitOps: GitOpsSpec{
				Enabled:      true,
				Tool:         "argocd",
				ImageUpdater: true,
				SyncWaves:    true,
			},
			Observability: ObservabilitySpec{
				Tracing: true,
				Metrics: true,
				Logging: true,
			},
			AI: AISpec{
				ClaudeCode: true,
				ClaudeMD:   true,
			},
		},
	}
}

// Presets defines named configuration presets.
var Presets = map[string][]string{
	"minimal":  {},
	"standard": {"redis", "kafka", "hasura", "external-secrets"},
	"full":     {"redis", "kafka", "hasura", "dapr", "observability", "external-secrets", "argocd", "langfuse"},
}
