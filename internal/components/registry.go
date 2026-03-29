// Package components defines the infrastructure component registry and dependency resolution.
package components

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// TemplateMapping maps a template source directory to its output subdirectory.
type TemplateMapping struct {
	Source string // Template directory in the embedded FS (e.g., "components/argocd/apps")
	Output string // Output subdirectory relative to project root (e.g., "k8s/argocd/apps")
}

// TiltConfig describes how a component integrates with the Tilt development workflow.
// Components that declare HelmPath must have a Tiltfile.tmpl in their template directory.
type TiltConfig struct {
	HelmPath  string   // Helm chart output path (e.g., "k8s/helm/infra/kafka"). Empty = no Tilt Helm integration.
	SetupFn   string   // Starlark function name (e.g., "setup_kafka"). Required when HelmPath is set.
	Local     bool     // If true, deployed via local dev manifests (not Helm-based Tiltfile).
	Required  bool     // If true, always included in 'auto' profile resolution.
	InfraDeps []string // Dependencies on other infra components in Tilt resource ordering.
}

// Component describes an infrastructure component's metadata and dependencies.
type Component struct {
	Name            string            // Unique identifier (e.g., "kafka", "redis")
	Description     string            // Human-readable description
	Tier            int               // Deployment tier (lower = earlier, used for topological sort)
	Requires        []string          // Hard dependencies (auto-installed)
	Suggests        []string          // Soft dependencies (prompted)
	Conflicts       []string          // Mutually exclusive components
	Templates       []string          // Template directories rendered to k8s/helm/infra/<name>
	ExtraTemplates  []TemplateMapping // Additional templates with explicit output paths
	Skills          []string          // Claude Code skills to generate
	ClaudeMD        []string          // CLAUDE.md sections to include
	Tilt            TiltConfig        // Tilt integration config (single source of truth for registry.yaml)
}

// builtinComponents defines all available infrastructure components.
var builtinComponents = map[string]*Component{
	"kafka": {
		Name:        "kafka",
		Description: "Apache Kafka event streaming via Strimzi operator",
		Tier:        0,
		Requires:    []string{},
		Suggests:    []string{"observability"},
		Templates:   []string{"components/kafka/helm"},
		Skills:      []string{"new-realtime-event"},
		ClaudeMD:    []string{"events"},
		Tilt:        TiltConfig{HelmPath: "k8s/helm/infra/kafka", SetupFn: "setup_kafka"},
	},
	"hasura": {
		Name:        "hasura",
		Description: "Hasura GraphQL engine with automatic API generation",
		Tier:        2,
		Requires:    []string{},
		Suggests:    []string{"kafka"},
		Templates:   []string{"components/hasura/helm"},
		ExtraTemplates: []TemplateMapping{
			{Source: "components/hasura/root", Output: "."},
			{Source: "components/hasura/project", Output: "hasura"},
		},
		Skills:   []string{"setup-graphql", "new-migration"},
		ClaudeMD: []string{"graphql"},
		Tilt:     TiltConfig{HelmPath: "k8s/helm/infra/hasura", SetupFn: "setup_hasura", InfraDeps: []string{"postgres"}},
	},
	"dapr": {
		Name:        "dapr",
		Description: "Dapr distributed application runtime for pub/sub and state",
		Tier:        1,
		Requires:    []string{},
		Suggests:    []string{"kafka"},
		Templates:   []string{"components/dapr/helm"},
		Tilt:        TiltConfig{HelmPath: "k8s/helm/infra/dapr", SetupFn: "setup_dapr"},
	},
	"redis": {
		Name:        "redis",
		Description: "Redis for caching and session storage",
		Tier:        0,
		Requires:    []string{},
		Templates:   []string{"components/redis/helm"},
		Skills:      []string{"setup-cache"},
		ClaudeMD:    []string{"caching"},
		Tilt:        TiltConfig{Local: true},
	},
	"observability": {
		Name:        "observability",
		Description: "Observability stack: OTEL collector, Tempo, Prometheus, Grafana",
		Tier:        0,
		Requires:    []string{},
		Templates:   []string{"components/observability/helm"},
		Tilt:        TiltConfig{HelmPath: "k8s/helm/infra/observability", SetupFn: "setup_observability_stack"},
	},
	"external-secrets": {
		Name:        "external-secrets",
		Description: "External Secrets Operator for syncing cloud secrets to K8s",
		Tier:        -1,
		Requires:    []string{},
		Templates:   []string{"components/external-secrets/helm"},
		Skills:      []string{"add-secret-group"},
	},
	"argocd": {
		Name:        "argocd",
		Description: "ArgoCD GitOps continuous deployment",
		Tier:        3,
		Requires:    []string{"external-secrets"},
		Templates:   []string{},
		ExtraTemplates: []TemplateMapping{
			{Source: "components/argocd/charts", Output: "k8s/argocd/charts/apps"},
			{Source: "components/argocd/apps", Output: "k8s/argocd/apps"},
			{Source: "components/argocd/scripts", Output: "k8s/argocd/scripts"},
		},
		Skills: []string{"setup-argocd"},
	},
	"langfuse": {
		Name:        "langfuse",
		Description: "Langfuse LLM observability platform",
		Tier:        2,
		Requires:    []string{"redis"},
		Templates:   []string{"components/langfuse/helm"},
		Tilt:        TiltConfig{HelmPath: "k8s/helm/infra/langfuse", SetupFn: "setup_langfuse", InfraDeps: []string{"redis", "postgres"}},
	},
	"k8s-dashboard": {
		Name:        "k8s-dashboard",
		Description: "Kubernetes Dashboard for cluster visibility",
		Tier:        0,
		Requires:    []string{},
		Templates:   []string{"components/k8s-dashboard/helm"},
	},
	"verdaccio": {
		Name:        "verdaccio",
		Description: "Private NPM registry for internal packages",
		Tier:        0,
		Requires:    []string{},
		Templates:   []string{"components/verdaccio/helm"},
	},
	"hasura-event-relay": {
		Name:        "hasura-event-relay",
		Description: "Hasura event relay: publishes Hasura events to Kafka via Dapr",
		Tier:        2,
		Requires:    []string{"hasura", "kafka", "dapr"},
		Templates:   []string{"components/hasura-event-relay/helm"},
	},
}

// Get returns a component by name, or nil if not found.
func Get(name string) *Component {
	return builtinComponents[name]
}

// List returns all available component names sorted alphabetically.
func List() []string {
	names := make([]string, 0, len(builtinComponents))
	for name := range builtinComponents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// All returns all component definitions.
func All() map[string]*Component {
	return builtinComponents
}

// ResolveDependencies takes a list of requested components and returns
// the full list including transitive dependencies, sorted by tier.
func ResolveDependencies(requested []string) ([]string, error) {
	resolved := make(map[string]bool)
	var order []string

	var resolve func(name string) error
	resolve = func(name string) error {
		if resolved[name] {
			return nil
		}

		comp := builtinComponents[name]
		if comp == nil {
			return fmt.Errorf("unknown component %q", name)
		}

		// Resolve hard dependencies first
		for _, dep := range comp.Requires {
			if err := resolve(dep); err != nil {
				return err
			}
		}

		resolved[name] = true
		order = append(order, name)
		return nil
	}

	for _, name := range requested {
		if err := resolve(name); err != nil {
			return nil, err
		}
	}

	// Sort by tier for deployment ordering
	sort.SliceStable(order, func(i, j int) bool {
		ci := builtinComponents[order[i]]
		cj := builtinComponents[order[j]]
		return ci.Tier < cj.Tier
	})

	return order, nil
}

// SkillsForComponents returns the union of skills needed for the given components.
func SkillsForComponents(componentNames []string) []string {
	seen := make(map[string]bool)
	var skills []string

	for _, name := range componentNames {
		comp := builtinComponents[name]
		if comp == nil {
			continue
		}
		for _, skill := range comp.Skills {
			if !seen[skill] {
				seen[skill] = true
				skills = append(skills, skill)
			}
		}
	}

	return skills
}

// ClaudeMDSections returns the CLAUDE.md sections needed for the given components.
func ClaudeMDSections(componentNames []string) []string {
	seen := make(map[string]bool)
	var sections []string

	for _, name := range componentNames {
		comp := builtinComponents[name]
		if comp == nil {
			continue
		}
		for _, section := range comp.ClaudeMD {
			if !seen[section] {
				seen[section] = true
				sections = append(sections, section)
			}
		}
	}

	return sections
}

// ValidateTemplates checks that every component with a Tilt HelmPath has a
// corresponding Tiltfile.tmpl in its template directory. This prevents the
// class of bugs where registry.yaml references a Tiltfile that doesn't exist.
//
// Call this in tests with the embedded templates.FS to catch missing files at build time.
func ValidateTemplates(templateFS fs.FS) error {
	var errs []string

	for name, comp := range builtinComponents {
		// Validate: if HelmPath is set, SetupFn must also be set
		if comp.Tilt.HelmPath != "" && comp.Tilt.SetupFn == "" {
			errs = append(errs, fmt.Sprintf(
				"component %q has Tilt.HelmPath=%q but no Tilt.SetupFn", name, comp.Tilt.HelmPath))
		}

		// Validate: if HelmPath is set, a Tiltfile.tmpl must exist in the template dir
		if comp.Tilt.HelmPath != "" {
			for _, tmplDir := range comp.Templates {
				tiltfile := tmplDir + "/Tiltfile.tmpl"
				if _, err := fs.Stat(templateFS, tiltfile); err != nil {
					errs = append(errs, fmt.Sprintf(
						"component %q declares Tilt.HelmPath=%q but is missing %s",
						name, comp.Tilt.HelmPath, tiltfile))
				}
			}
		}

		// Validate: Templates dirs exist
		for _, tmplDir := range comp.Templates {
			if _, err := fs.Stat(templateFS, tmplDir); err != nil {
				errs = append(errs, fmt.Sprintf(
					"component %q references template dir %q that doesn't exist", name, tmplDir))
			}
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("component template validation failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// InfraRegistryEntry represents a single entry in the generated tilt/registry.yaml.
type InfraRegistryEntry struct {
	Name      string
	HelmPath  string
	SetupFn   string
	Local     bool
	Required  bool
	InfraDeps []string
}

// InfraRegistryEntries returns the Tilt registry entries for the given components,
// suitable for rendering in registry.yaml.tmpl.
func InfraRegistryEntries(componentNames []string) []InfraRegistryEntry {
	var entries []InfraRegistryEntry
	for _, name := range componentNames {
		comp := builtinComponents[name]
		if comp == nil {
			continue
		}
		t := comp.Tilt
		// Only include components that have Tilt config (helmPath, local, or required)
		if t.HelmPath == "" && !t.Local && !t.Required {
			continue
		}
		entries = append(entries, InfraRegistryEntry{
			Name:      name,
			HelmPath:  t.HelmPath,
			SetupFn:   t.SetupFn,
			Local:     t.Local,
			Required:  t.Required,
			InfraDeps: t.InfraDeps,
		})
	}
	return entries
}
