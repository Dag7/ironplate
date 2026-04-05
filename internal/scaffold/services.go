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
	"github.com/dag7/ironplate/pkg/fsutil"
)


// ExampleService defines an example service to generate during init.
type ExampleService struct {
	Name      string
	Type      string // "node-api", "go-api", "nextjs"
	Group     string
	Port      int // Host-side Tilt port-forward (container always listens on 3000)
	DebugPort int // Host-side debug port-forward (container: 9229 node, 40000 go)
	Features  []string
}

// DefaultExampleServices returns the example services based on the project's language selection.
// All containers listen on port 3000 (HTTP) and 9229 (node debug) / 40000 (go debug).
// The Port/DebugPort here are host-side Tilt port-forwards, incrementally assigned.
func DefaultExampleServices(cfg *config.ProjectConfig) []ExampleService {
	pa := newPortAllocator()

	var services []ExampleService

	if cfg.Spec.HasLanguage("node") {
		services = append(services,
			pa.allocate("api", "node-api", "core"),
			pa.allocate("web", "nextjs", "frontend"),
		)
	}

	if cfg.Spec.HasLanguage("go") {
		name := "api"
		if cfg.Spec.HasLanguage("node") {
			name = "api-go"
		}
		services = append(services, pa.allocate(name, "go-api", "core"))
	}

	return services
}

// portAllocator tracks port assignments to avoid collisions.
type portAllocator struct {
	nextPort      int
	nextNodeDebug int
	nextGoDebug   int
}

func newPortAllocator() *portAllocator {
	return &portAllocator{
		nextPort:      BaseForwardPort,
		nextNodeDebug: BaseNodeDebugPort,
		nextGoDebug:   BaseGoDebugPort,
	}
}

func (pa *portAllocator) allocate(name, serviceType, group string) ExampleService {
	svc := ExampleService{
		Name:  name,
		Type:  serviceType,
		Group: group,
		Port:  pa.nextPort,
	}
	pa.nextPort++

	if isGoService(serviceType) {
		svc.DebugPort = pa.nextGoDebug
		pa.nextGoDebug++
	} else {
		svc.DebugPort = pa.nextNodeDebug
		pa.nextNodeDebug++
	}

	return svc
}

// RenderServiceParams holds parameters for the shared service rendering pipeline.
type RenderServiceParams struct {
	Renderer    *engine.Renderer
	Templates   fs.FS
	ProjectRoot string
	Config      *config.ProjectConfig
	Ctx         *engine.TemplateContext
}

// RenderService executes the core service generation pipeline shared by both
// `iron generate service` and `iron init --example-services`:
//  1. Render service source files (excluding helm/)
//  2. Create or update the group umbrella Helm chart
//  3. Register the service in the Tilt registry
//  4. Register with ArgoCD (if enabled)
//
// It returns non-fatal warnings as a string slice; only fatal errors return an error.
func RenderService(p RenderServiceParams) (warnings []string, err error) {
	svc := p.Ctx.Service
	templateDir, ok := config.ServiceTemplateDirs[svc.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported service type %q", svc.Type)
	}

	// 1. Render service source files (exclude helm/ — chart is handled in step 2)
	serviceDir := filepath.Join(p.ProjectRoot, "apps", svc.Name)
	if err := p.Renderer.RenderFS(p.Templates, templateDir, serviceDir, p.Ctx, "helm"); err != nil {
		return nil, fmt.Errorf("render service files for %s: %w", svc.Name, err)
	}

	// 2. Create or update group umbrella Helm chart
	groupChartDir := filepath.Join(p.ProjectRoot, "k8s", "helm", p.Config.Metadata.Name, svc.Group)
	chartYAML := filepath.Join(groupChartDir, "Chart.yaml")
	if !fsutil.FileExists(chartYAML) {
		if err := p.Renderer.RenderFS(p.Templates, "service/helm", groupChartDir, p.Ctx); err != nil {
			return nil, fmt.Errorf("render umbrella chart for group %s: %w", svc.Group, err)
		}
	} else {
		if err := AppendServiceToUmbrellaValues(groupChartDir, svc); err != nil {
			return nil, fmt.Errorf("update umbrella values for %s: %w", svc.Name, err)
		}
	}

	// 3. Register in Tilt registry
	if err := RegisterServiceInRegistry(p.ProjectRoot, p.Ctx); err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not update Tilt registry: %s", err))
	}

	// 4. Register in centralized ingress chart
	if err := RegisterServiceIngress(p.ProjectRoot, p.Config, p.Ctx); err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not update ingress: %s", err))
	}

	// 5. Register with ArgoCD (if enabled)
	if p.Config.Spec.GitOps.Enabled && p.Config.Spec.Infrastructure.HasComponent("argocd") {
		if err := RegisterArgoCDService(p.ProjectRoot, p.Config, p.Ctx); err != nil {
			warnings = append(warnings, fmt.Sprintf("Could not update ArgoCD: %s", err))
		}
	}

	return warnings, nil
}

// GenerateExampleServices generates example services in a scaffolded project.
// Called after the main scaffold pipeline to add starter services.
func GenerateExampleServices(cfg *config.ProjectConfig, outputDir string, tmplFS fs.FS, services []ExampleService) error {
	renderer := engine.NewRenderer()

	fmt.Println()
	fmt.Println("  Generating example services")

	for i, svc := range services {
		fmt.Printf("  [%d/%d] Creating %s (%s)\n", i+1, len(services), svc.Name, svc.Type)

		ctx := engine.NewTemplateContext(cfg)
		ctx.Service = &engine.ServiceTemplateData{
			Name:      svc.Name,
			Type:      svc.Type,
			Group:     svc.Group,
			Port:      svc.Port,
			DebugPort: svc.DebugPort,
			SrcFolder: "apps",
			Features:  svc.Features,
		}

		warnings, err := RenderService(RenderServiceParams{
			Renderer:    renderer,
			Templates:   tmplFS,
			ProjectRoot: outputDir,
			Config:      cfg,
			Ctx:         ctx,
		})
		if err != nil {
			return fmt.Errorf("generate service %s: %w", svc.Name, err)
		}
		for _, w := range warnings {
			fmt.Printf("  ! %s\n", w)
		}

		cfg.Spec.Services = append(cfg.Spec.Services, config.ServiceSpec{
			Name:     svc.Name,
			Type:     svc.Type,
			Group:    svc.Group,
			Port:     svc.Port,
			Features: svc.Features,
		})
	}

	// Save updated config using the canonical Save path
	cfgPath := filepath.Join(outputDir, "ironplate.yaml")
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("  ✓ Created %d example service(s)\n", len(services))
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

	// Helm port is always 3000 (container port), not the host-side forward port
	entry := fmt.Sprintf("\n  %s:\n    enabled: true\n    port: 3000\n    image:\n      tag: latest\n    env:\n      LOG_LEVEL: debug\n",
		svc.Name)

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

// RegisterServiceIngress adds a service ingress entry to the centralized ingress
// chart's values.yaml. Each service gets a {name}.localhost route.
func RegisterServiceIngress(projectRoot string, cfg *config.ProjectConfig, ctx *engine.TemplateContext) error {
	svc := ctx.Service
	valuesPath := filepath.Join(projectRoot, "k8s", "helm", cfg.Metadata.Name, "ingress", "values.yaml")

	if !fsutil.FileExists(valuesPath) {
		return nil // ingress chart not scaffolded yet
	}

	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("read ingress values: %w", err)
	}

	content := string(data)

	// Check if this service already has an ingress entry (use host to avoid
	// matching other YAML keys like middlewarePresets.api)
	marker := fmt.Sprintf("host: %s.localhost", svc.Name)
	if strings.Contains(content, marker) {
		return nil
	}

	// Determine the middleware preset based on service type
	preset := "api"
	if svc.Type == "nextjs" {
		preset = "default"
	}

	// Build the ingress entry (port 80 = k8s Service port, not container port)
	entry := fmt.Sprintf("  %s:\n    enabled: true\n    host: %s.localhost\n    middlewarePreset: %s\n    routes:\n      - path: /\n        service: %s\n        port: 80\n",
		svc.Name, svc.Name, preset, svc.Name)

	// Replace empty `ingresses: {}` or append after `ingresses:` line
	if strings.Contains(content, "ingresses: {}") {
		content = strings.Replace(content, "ingresses: {}", "ingresses:\n"+entry, 1)
	} else if strings.Contains(content, "ingresses:\n") {
		// Find the ingresses block and append before infraIngresses
		infraIdx := strings.Index(content, "\ninfraIngresses:")
		if infraIdx > 0 {
			content = content[:infraIdx] + "\n" + entry + content[infraIdx:]
		}
	}

	return os.WriteFile(valuesPath, []byte(content), 0o644)
}
