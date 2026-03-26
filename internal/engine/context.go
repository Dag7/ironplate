package engine

import (
	"time"

	"github.com/dag7/ironplate/internal/config"
)

// TemplateContext is the root context passed to all templates.
type TemplateContext struct {
	// Project is the full project configuration.
	Project *config.ProjectConfig

	// Computed values derived from configuration.
	Computed ComputedValues

	// GoModule is the Go module path (e.g., "github.com/org/project").
	GoModule string

	// Service is set when rendering service-specific templates.
	Service *ServiceTemplateData

	// Package is set when rendering package-specific templates.
	Package *PackageTemplateData
}

// ServiceTemplateData holds service-specific template data.
type ServiceTemplateData struct {
	Name      string
	Type      string // "node-api" | "go-api" | "nextjs"
	Group     string // Helm group
	Port      int
	DebugPort int
	SrcFolder string // "apps"
	Features  []string
}

// HasFeature checks if the service has a specific feature enabled.
func (s *ServiceTemplateData) HasFeature(feature string) bool {
	for _, f := range s.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// PackageTemplateData holds package-specific template data.
type PackageTemplateData struct {
	Name       string
	NamePascal string
	NameSnake  string
	Language   string // "node" | "go"
	Scope      string // "@myproject"
}

// ComputedValues holds values derived from configuration for template convenience.
type ComputedValues struct {
	// NamePascal is the project name in PascalCase.
	NamePascal string
	// NameCamel is the project name in camelCase.
	NameCamel string
	// NameSnake is the project name in snake_case.
	NameSnake string

	// Aliases for backward compatibility.
	ProjectNamePascal string
	ProjectNameCamel  string
	ProjectNameSnake  string

	// PrimaryScope is the first scope (e.g., "@myproject").
	PrimaryScope string

	// HasNode indicates if Node.js is a project language.
	HasNode bool
	// HasGo indicates if Go is a project language.
	HasGo bool

	// Component flags for conditional template rendering.
	HasKafka           bool
	HasHasura          bool
	HasDapr            bool
	HasRedis           bool
	HasObservability   bool
	HasExternalSecrets bool
	HasArgoCD          bool
	HasLangfuse        bool

	// Year for copyright and license templates.
	Year int
}

// UpdateComputedComponents refreshes component flags from a resolved list
// (which may include auto-resolved dependencies not in the original config).
func (ctx *TemplateContext) UpdateComputedComponents(resolved []string) {
	has := make(map[string]bool, len(resolved))
	for _, c := range resolved {
		has[c] = true
	}
	ctx.Computed.HasKafka = has["kafka"]
	ctx.Computed.HasHasura = has["hasura"]
	ctx.Computed.HasDapr = has["dapr"]
	ctx.Computed.HasRedis = has["redis"]
	ctx.Computed.HasObservability = has["observability"]
	ctx.Computed.HasExternalSecrets = has["external-secrets"]
	ctx.Computed.HasArgoCD = has["argocd"]
	ctx.Computed.HasLangfuse = has["langfuse"]
}

// NewTemplateContext creates a TemplateContext from a ProjectConfig.
func NewTemplateContext(cfg *config.ProjectConfig) *TemplateContext {
	infra := cfg.Spec.Infrastructure

	scope := "@" + cfg.Metadata.Organization
	if len(cfg.Spec.Monorepo.Scopes) > 0 {
		scope = cfg.Spec.Monorepo.Scopes[0]
	}

	namePascal := toPascalCase(cfg.Metadata.Name)
	nameCamel := toCamelCase(cfg.Metadata.Name)
	nameSnake := toSnakeCase(cfg.Metadata.Name)

	goModule := "github.com/" + cfg.Metadata.Organization + "/" + cfg.Metadata.Name

	return &TemplateContext{
		Project:  cfg,
		GoModule: goModule,
		Computed: ComputedValues{
			NamePascal:        namePascal,
			NameCamel:         nameCamel,
			NameSnake:         nameSnake,
			ProjectNamePascal: namePascal,
			ProjectNameCamel:  nameCamel,
			ProjectNameSnake:  nameSnake,
			PrimaryScope:      scope,

			HasNode: cfg.Spec.HasLanguage("node"),
			HasGo:   cfg.Spec.HasLanguage("go"),

			HasKafka:           infra.HasComponent("kafka"),
			HasHasura:          infra.HasComponent("hasura"),
			HasDapr:            infra.HasComponent("dapr"),
			HasRedis:           infra.HasComponent("redis"),
			HasObservability:   infra.HasComponent("observability"),
			HasExternalSecrets: infra.HasComponent("external-secrets"),
			HasArgoCD:          infra.HasComponent("argocd"),
			HasLangfuse:        infra.HasComponent("langfuse"),

			Year: time.Now().Year(),
		},
	}
}
