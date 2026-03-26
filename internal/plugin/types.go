// Package plugin defines the core plugin interfaces following SOLID principles.
//
// Design Patterns Used:
//   - Strategy: CloudProvider implementations are interchangeable (GCP, AWS, Azure)
//   - Template Method: ServiceGenerator defines a shared generation algorithm with variable steps
//   - Registry: Plugins are registered and discovered by name
//   - Factory: Plugin creation is deferred to registration time
package plugin

// Plugin is the base interface for all ironplate plugins.
type Plugin interface {
	// Name returns the unique identifier for this plugin.
	Name() string
	// Description returns a human-readable description.
	Description() string
}

// ProjectContext carries configuration data through the generation pipeline.
type ProjectContext struct {
	Config    interface{} // *config.ProjectConfig
	OutputDir string
	DryRun    bool
}

// ServiceContext carries service-specific data for generation.
type ServiceContext struct {
	ProjectContext
	ServiceName string
	ServiceType string
	Group       string
	Port        int
	Features    []string
}

// PackageContext carries package-specific data for generation.
type PackageContext struct {
	ProjectContext
	PackageName string
	Scope       string
	Language    string
}

// RegistryConfig holds container registry configuration from a cloud provider.
type RegistryConfig struct {
	Type string // e.g., "artifact-registry"
	URL  string // e.g., "us-central1-docker.pkg.dev/myproject/repo"
}

// CloudProvider abstracts cloud-specific infrastructure generation.
// Implements the Strategy pattern: swap GCP/AWS/Azure without modifying core code.
type CloudProvider interface {
	Plugin

	// GenerateIaC generates Infrastructure-as-Code files (Pulumi, Terraform, etc.)
	GenerateIaC(ctx *ProjectContext) error

	// GenerateCIAuth generates CI/CD authentication configuration.
	GenerateCIAuth(ctx *ProjectContext) error

	// RegistryConfig returns container registry configuration.
	RegistryConfig() RegistryConfig

	// RequiredAPIs returns cloud APIs that must be enabled.
	RequiredAPIs() []string

	// SupportedComponents returns infrastructure components this provider supports.
	SupportedComponents() []string
}

// ServiceGenerator generates service boilerplate.
// Implements the Template Method pattern: shared algorithm with variable steps per language.
type ServiceGenerator interface {
	Plugin

	// Language returns the programming language (e.g., "node", "go", "nextjs").
	Language() string

	// ServiceType returns the service type (e.g., "api", "worker", "frontend").
	ServiceType() string

	// Generate creates the service files.
	Generate(ctx *ServiceContext) error

	// HelmFragment returns Helm values to merge for this service.
	HelmFragment(ctx *ServiceContext) map[string]interface{}

	// TiltFragment returns a Tiltfile snippet for this service.
	TiltFragment(ctx *ServiceContext) string
}

// PackageGenerator generates shared package boilerplate.
type PackageGenerator interface {
	Plugin

	// Language returns the target language.
	Language() string

	// Generate creates the package files.
	Generate(ctx *PackageContext) error
}

// InfraComponent represents an infrastructure component (Kafka, Redis, etc.).
// Registered in the component registry for extensible infrastructure management.
type InfraComponent interface {
	Plugin

	// Tier returns the deployment tier (lower = deployed first).
	Tier() int

	// DependsOn returns names of components this one requires.
	DependsOn() []string

	// GenerateHelm generates Helm chart files for this component.
	GenerateHelm(ctx *ProjectContext) error

	// TiltSetup returns a Tiltfile snippet for local development.
	TiltSetup() string

	// DefaultConfig returns default configuration values.
	DefaultConfig() map[string]interface{}
}
