package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/engine"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/dag7/ironplate/pkg/fsutil"
)

// serviceTemplateDirs maps service types to their template directories.
var serviceTemplateDirs = map[string]string{
	"node-api": "service/node",
	"go-api":   "service/go",
	"nextjs":   "service/nextjs",
}

// ExampleService defines an example service to generate during init.
type ExampleService struct {
	Name     string
	Type     string // "node-api", "go-api", "nextjs"
	Group    string
	Port     int
	Features []string
}

// DefaultExampleServices returns the example services based on the project's language selection.
func DefaultExampleServices(cfg *config.ProjectConfig) []ExampleService {
	var services []ExampleService

	if cfg.Spec.HasLanguage("node") {
		services = append(services, ExampleService{
			Name:  "api",
			Type:  "node-api",
			Group: "core",
			Port:  3000,
		})
		services = append(services, ExampleService{
			Name:  "web",
			Type:  "nextjs",
			Group: "frontend",
			Port:  3100,
		})
	}

	if cfg.Spec.HasLanguage("go") {
		name := "api"
		port := 3000
		if cfg.Spec.HasLanguage("node") {
			name = "api-go" // Avoid collision with node-api service
			port = 3200     // Don't conflict with node services
		}
		services = append(services, ExampleService{
			Name:  name,
			Type:  "go-api",
			Group: "core",
			Port:  port,
		})
	}

	return services
}

// GenerateExampleServices generates example services in a scaffolded project.
// Called after the main scaffold pipeline to add starter services.
func GenerateExampleServices(cfg *config.ProjectConfig, outputDir string, templates fs.FS, services []ExampleService) error {
	printer := tui.NewStatusPrinter()
	renderer := engine.NewRenderer()

	printer.Section("Generating example services")

	for i, svc := range services {
		templateDir, ok := serviceTemplateDirs[svc.Type]
		if !ok {
			printer.Warning(fmt.Sprintf("Unknown service type %q, skipping", svc.Type))
			continue
		}

		printer.Step(i+1, len(services), fmt.Sprintf("Creating %s (%s)", svc.Name, svc.Type))

		ctx := engine.NewTemplateContext(cfg)
		ctx.Service = &engine.ServiceTemplateData{
			Name:      svc.Name,
			Type:      svc.Type,
			Group:     svc.Group,
			Port:      svc.Port,
			DebugPort: 9229 + i,
			SrcFolder: "apps",
			Features:  svc.Features,
		}

		// 1. Render service source files (exclude helm/)
		serviceDir := filepath.Join(outputDir, "apps", svc.Name)
		if err := renderer.RenderFS(templates, templateDir, serviceDir, ctx, "helm"); err != nil {
			return fmt.Errorf("render service %s: %w", svc.Name, err)
		}

		// 2. Create group umbrella Helm chart
		groupChartDir := filepath.Join(outputDir, "k8s", "helm", cfg.Metadata.Name, svc.Group)
		chartYAML := filepath.Join(groupChartDir, "Chart.yaml")
		if !fsutil.FileExists(chartYAML) {
			if err := renderer.RenderFS(templates, "service/helm", groupChartDir, ctx); err != nil {
				return fmt.Errorf("render umbrella chart for %s: %w", svc.Group, err)
			}
		} else {
			if err := AppendServiceToUmbrellaValues(groupChartDir, ctx.Service); err != nil {
				return fmt.Errorf("update umbrella values for %s: %w", svc.Name, err)
			}
		}

		// 3. Register in Tilt registry
		if err := RegisterServiceInRegistry(outputDir, ctx); err != nil {
			printer.Warning(fmt.Sprintf("Could not update Tilt registry: %s", err))
		}

		// 4. Register with ArgoCD (if enabled)
		if cfg.Spec.GitOps.Enabled && cfg.Spec.Infrastructure.HasComponent("argocd") {
			if err := RegisterArgoCDService(outputDir, cfg, ctx); err != nil {
				printer.Warning(fmt.Sprintf("Could not update ArgoCD: %s", err))
			}
		}

		// 5. Add to config services list
		cfg.Spec.Services = append(cfg.Spec.Services, config.ServiceSpec{
			Name:     svc.Name,
			Type:     svc.Type,
			Group:    svc.Group,
			Port:     svc.Port,
			Features: svc.Features,
		})
	}

	// Write updated ironplate.yaml
	cfgPath := filepath.Join(outputDir, "ironplate.yaml")
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	printer.Success(fmt.Sprintf("Created %d example service(s)", len(services)))
	return nil
}

// registryFile is the path to the Tilt service registry relative to project root.
const registryFile = "tilt/registry.yaml"

// tiltRegistry represents the structure of tilt/registry.yaml.
type tiltRegistry struct {
	Services       map[string]tiltServiceEntry `yaml:"services"`
	Infrastructure map[string]interface{}      `yaml:"infrastructure"`
}

// tiltServiceEntry represents a single service in the Tilt registry.
type tiltServiceEntry struct {
	Type      string           `yaml:"type"`
	Group     string           `yaml:"group"`
	Port      int              `yaml:"port"`
	DebugPort int              `yaml:"debugPort,omitempty"`
	Src       string           `yaml:"src"`
	Labels    []string         `yaml:"labels,omitempty"`
	Deps      *tiltServiceDeps `yaml:"deps,omitempty"`
}

// tiltServiceDeps represents service dependency declarations.
type tiltServiceDeps struct {
	Infra    []string `yaml:"infra,omitempty"`
	Services []string `yaml:"services,omitempty"`
}

// RegisterServiceInRegistry adds a service entry to tilt/registry.yaml.
func RegisterServiceInRegistry(projectRoot string, ctx *engine.TemplateContext) error {
	regPath := filepath.Join(projectRoot, registryFile)
	svc := ctx.Service

	var reg tiltRegistry
	if fsutil.FileExists(regPath) {
		data, err := os.ReadFile(regPath)
		if err != nil {
			return fmt.Errorf("read registry: %w", err)
		}
		if err := yaml.Unmarshal(data, &reg); err != nil {
			return fmt.Errorf("parse registry: %w", err)
		}
	}

	if reg.Services == nil {
		reg.Services = make(map[string]tiltServiceEntry)
	}

	if _, exists := reg.Services[svc.Name]; exists {
		return nil
	}

	category := "backend"
	if svc.Type == "nextjs" {
		category = "frontend"
	}
	labels := []string{category}
	if svc.Group != category {
		labels = append(labels, svc.Group)
	}

	infraDepsMap := make(map[string]bool)
	for _, f := range svc.Features {
		switch f {
		case "hasura":
			infraDepsMap["hasura"] = true
		case "cache":
			infraDepsMap["redis"] = true
		case "dapr", "eventbus":
			infraDepsMap["kafka"] = true
		}
	}
	infraDeps := make([]string, 0, len(infraDepsMap))
	for dep := range infraDepsMap {
		infraDeps = append(infraDeps, dep)
	}
	sort.Strings(infraDeps)

	entry := tiltServiceEntry{
		Type:      svc.Type,
		Group:     svc.Group,
		Port:      svc.Port,
		DebugPort: svc.DebugPort,
		Src:       fmt.Sprintf("%s/%s", svc.SrcFolder, svc.Name),
		Labels:    labels,
	}
	if len(infraDeps) > 0 {
		entry.Deps = &tiltServiceDeps{Infra: infraDeps}
	}

	reg.Services[svc.Name] = entry

	out, err := yaml.Marshal(&reg)
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(regPath), 0o755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	return os.WriteFile(regPath, out, 0o644)
}

// AppendServiceToUmbrellaValues appends a new service entry to the group umbrella
// chart's values.yaml.
func AppendServiceToUmbrellaValues(groupChartDir string, svc *engine.ServiceTemplateData) error {
	valuesPath := filepath.Join(groupChartDir, "values.yaml")

	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("read values.yaml: %w", err)
	}

	content := string(data)
	if strings.Contains(content, fmt.Sprintf("  %s:", svc.Name)) {
		return nil
	}

	entry := fmt.Sprintf("\n  %s:\n    enabled: true\n    port: %d\n    image:\n      tag: latest\n    env:\n      LOG_LEVEL: debug\n",
		svc.Name, svc.Port)

	content += entry
	return os.WriteFile(valuesPath, []byte(content), 0o644)
}

// argoCDValuesFile is the path to the ArgoCD app-of-apps values file.
const argoCDValuesFile = "k8s/argocd/charts/apps/values.yaml"

type argoCDServiceGroups struct {
	ServiceGroups map[string]*argoCDGroupEntry `yaml:"serviceGroups"`
}

type argoCDGroupEntry struct {
	SyncWave  string              `yaml:"syncWave"`
	ChartPath string              `yaml:"chartPath"`
	Services  []argoCDServiceName `yaml:"services"`
}

type argoCDServiceName struct {
	Name string `yaml:"name"`
}

// RegisterArgoCDService registers the service in the ArgoCD app-of-apps values.
func RegisterArgoCDService(projectRoot string, cfg *config.ProjectConfig, ctx *engine.TemplateContext) error {
	svc := ctx.Service
	valuesPath := filepath.Join(projectRoot, argoCDValuesFile)

	if !fsutil.FileExists(valuesPath) {
		return nil // ArgoCD not scaffolded yet
	}

	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("read ArgoCD values: %w", err)
	}

	var fullValues map[string]interface{}
	if err := yaml.Unmarshal(data, &fullValues); err != nil {
		return fmt.Errorf("parse ArgoCD values: %w", err)
	}

	var groups argoCDServiceGroups
	if err := yaml.Unmarshal(data, &groups); err != nil {
		return fmt.Errorf("parse ArgoCD serviceGroups: %w", err)
	}

	if groups.ServiceGroups == nil {
		groups.ServiceGroups = make(map[string]*argoCDGroupEntry)
	}

	groupEntry, exists := groups.ServiceGroups[svc.Group]
	if exists {
		for _, s := range groupEntry.Services {
			if s.Name == svc.Name {
				return nil
			}
		}
		groupEntry.Services = append(groupEntry.Services, argoCDServiceName{Name: svc.Name})
	} else {
		groups.ServiceGroups[svc.Group] = &argoCDGroupEntry{
			SyncWave:  "4",
			ChartPath: svc.Group,
			Services:  []argoCDServiceName{{Name: svc.Name}},
		}
	}

	fullValues["serviceGroups"] = groups.ServiceGroups

	out, err := yaml.Marshal(fullValues)
	if err != nil {
		return fmt.Errorf("marshal ArgoCD values: %w", err)
	}

	return os.WriteFile(valuesPath, out, 0o644)
}
