package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/engine"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/dag7/ironplate/pkg/fsutil"
	"github.com/dag7/ironplate/templates"
)

// serviceTemplateDirs maps service types to their template directories.
// Add new service types here — no need to modify command logic.
var serviceTemplateDirs = map[string]string{
	"node-api": "service/node",
	"go-api":   "service/go",
	"nextjs":   "service/nextjs",
}

// registryFile is the path to the Tilt service registry relative to project root.
const registryFile = "tilt/registry.yaml"

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen", "g"},
		Short:   "Generate services, packages, and components",
		Long:    `Generate boilerplate for services, shared packages, and infrastructure components.`,
	}

	cmd.AddCommand(newGenerateServiceCmd())
	cmd.AddCommand(newGeneratePackageCmd())

	return cmd
}

func newGenerateServiceCmd() *cobra.Command {
	var (
		serviceType string
		group       string
		features    string
		port        int
	)

	cmd := &cobra.Command{
		Use:   "service <name>",
		Short: "Generate a new microservice",
		Long: `Generate a new microservice with source code, Helm chart, Tiltfile entry,
and optional ArgoCD registration. The service is fully wired into the monorepo.

Examples:
  iron generate service auth-service --type node-api --group auth
  iron generate service payment-worker --type go-api --features dapr,eventbus
  iron generate service web-app --type nextjs --group frontend --port 3100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceName := args[0]
			printer := tui.NewStatusPrinter()

			// Find and load project config
			cfgPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("not in an ironplate project: %w", err)
			}

			cfgData, err := os.ReadFile(cfgPath)
			if err != nil {
				return fmt.Errorf("read config: %w", err)
			}

			cfg, err := config.Parse(cfgData)
			if err != nil {
				return fmt.Errorf("parse config: %w", err)
			}

			projectRoot := filepath.Dir(cfgPath)

			// Check for duplicate service name
			for _, svc := range cfg.Spec.Services {
				if svc.Name == serviceName {
					return fmt.Errorf("service %q already exists in ironplate.yaml", serviceName)
				}
			}

			// Determine template type via registry lookup
			templateDir, ok := serviceTemplateDirs[serviceType]
			if !ok {
				supported := make([]string, 0, len(serviceTemplateDirs))
				for k := range serviceTemplateDirs {
					supported = append(supported, k)
				}
				return fmt.Errorf("unsupported service type %q — supported: %s", serviceType, strings.Join(supported, ", "))
			}

			// Default group
			if group == "" {
				group = "core"
			}

			// Default port allocation
			if port == 0 {
				port = 3000 + len(cfg.Spec.Services)
			}

			// Parse features
			var featureList []string
			if features != "" {
				featureList = strings.Split(features, ",")
			}

			// Build template context
			ctx := engine.NewTemplateContext(cfg)
			ctx.Service = &engine.ServiceTemplateData{
				Name:      serviceName,
				Type:      serviceType,
				Group:     group,
				Port:      port,
				DebugPort: 9229 + len(cfg.Spec.Services),
				SrcFolder: "apps",
				Features:  featureList,
			}

			renderer := engine.NewRenderer()
			totalSteps := 6
			step := 0

			printer.Section("Generating service: " + serviceName)

			// 1. Render service source files (exclude helm/ — chart is handled in step 2)
			step++
			serviceDir := filepath.Join(projectRoot, "apps", serviceName)
			printer.Step(step, totalSteps, "Creating service source files")
			if err := renderer.RenderFS(templates.FS, templateDir, serviceDir, ctx, "helm"); err != nil {
				return fmt.Errorf("render service files: %w", err)
			}

			// 2. Create or update group umbrella Helm chart
			step++
			groupChartDir := filepath.Join(projectRoot, "k8s", "helm", cfg.Metadata.Name, group)
			chartYAML := filepath.Join(groupChartDir, "Chart.yaml")
			if !fsutil.FileExists(chartYAML) {
				// First service in this group — render the umbrella chart template
				printer.Step(step, totalSteps, "Creating group umbrella chart")
				if err := renderer.RenderFS(templates.FS, "service/helm", groupChartDir, ctx); err != nil {
					return fmt.Errorf("render umbrella chart: %w", err)
				}
			} else {
				// Additional service — append to existing values.yaml
				printer.Step(step, totalSteps, "Adding service to group umbrella chart")
				if err := appendServiceToUmbrellaValues(groupChartDir, ctx.Service); err != nil {
					return fmt.Errorf("update umbrella values: %w", err)
				}
			}

			// 3. Register service in Tilt registry
			step++
			printer.Step(step, totalSteps, "Updating Tilt registry")
			if err := registerServiceInRegistry(projectRoot, ctx); err != nil {
				printer.Warning(fmt.Sprintf("Could not update registry: %s", err))
			}

			// 4. Register with ArgoCD
			step++
			printer.Step(step, totalSteps, "Registering with ArgoCD")
			if cfg.Spec.GitOps.Enabled && cfg.Spec.Infrastructure.HasComponent("argocd") {
				if err := registerArgoCDService(projectRoot, cfg, ctx); err != nil {
					printer.Warning(fmt.Sprintf("Could not update ArgoCD: %s", err))
				}
			} else {
				printer.Info("ArgoCD not enabled — skipped")
			}

			// 5. Update ironplate.yaml
			step++
			printer.Step(step, totalSteps, "Updating ironplate.yaml")
			cfg.Spec.Services = append(cfg.Spec.Services, config.ServiceSpec{
				Name:     serviceName,
				Type:     serviceType,
				Group:    group,
				Port:     port,
				Features: featureList,
			})

			cfgData, err = yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}
			if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			// 6. Summary
			step++
			printer.Step(step, totalSteps, "Done")
			fmt.Println()
			printer.Success(fmt.Sprintf("Service %s created successfully!", serviceName))
			printer.Info(fmt.Sprintf("Source:   apps/%s/", serviceName))
			printer.Info(fmt.Sprintf("Helm:     k8s/helm/%s/%s/", cfg.Metadata.Name, group))
			printer.Info(fmt.Sprintf("Port:     %d", port))
			if len(featureList) > 0 {
				printer.Info(fmt.Sprintf("Features: %s", strings.Join(featureList, ", ")))
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&serviceType, "type", "node-api", "Service type: node-api, go-api, nextjs")
	cmd.Flags().StringVar(&group, "group", "", "Helm chart group name")
	cmd.Flags().StringVar(&features, "features", "", "Comma-separated features: hasura, cache, dapr, eventbus")
	cmd.Flags().IntVar(&port, "port", 0, "Service port (auto-allocated if not set)")

	return cmd
}

// appendServiceToUmbrellaValues appends a new service entry to the group umbrella
// chart's values.yaml. Uses string append to preserve existing formatting/comments.
func appendServiceToUmbrellaValues(groupChartDir string, svc *engine.ServiceTemplateData) error {
	valuesPath := filepath.Join(groupChartDir, "values.yaml")

	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("read values.yaml: %w", err)
	}

	content := string(data)

	// Check if the service is already registered
	if strings.Contains(content, fmt.Sprintf("  %s:", svc.Name)) {
		return nil
	}

	entry := fmt.Sprintf("\n  %s:\n    enabled: true\n    port: %d\n    image:\n      tag: latest\n    env:\n      LOG_LEVEL: debug\n",
		svc.Name, svc.Port)

	content += entry

	return os.WriteFile(valuesPath, []byte(content), 0o644)
}

// tiltRegistry represents the structure of tilt/registry.yaml.
type tiltRegistry struct {
	Services       map[string]tiltServiceEntry `yaml:"services"`
	Infrastructure map[string]interface{}      `yaml:"infrastructure"`
}

// tiltServiceEntry represents a single service in the Tilt registry.
type tiltServiceEntry struct {
	Type      string            `yaml:"type"`
	Group     string            `yaml:"group"`
	Port      int               `yaml:"port"`
	DebugPort int               `yaml:"debugPort,omitempty"`
	Src       string            `yaml:"src"`
	Labels    []string          `yaml:"labels,omitempty"`
	Deps      *tiltServiceDeps  `yaml:"deps,omitempty"`
}

// tiltServiceDeps represents service dependency declarations.
type tiltServiceDeps struct {
	Infra    []string `yaml:"infra,omitempty"`
	Services []string `yaml:"services,omitempty"`
}

// registerServiceInRegistry adds or updates a service entry in tilt/registry.yaml.
func registerServiceInRegistry(projectRoot string, ctx *engine.TemplateContext) error {
	regPath := filepath.Join(projectRoot, registryFile)
	svc := ctx.Service

	// Read existing registry or create a new one
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

	// Check if service is already registered
	if _, exists := reg.Services[svc.Name]; exists {
		return nil
	}

	// Build labels from group and type
	labels := []string{"backend", svc.Group}
	if svc.Type == "nextjs" {
		labels = []string{"frontend", svc.Group}
	}

	// Build infra deps from features (deduplicated)
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

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(regPath), 0o755); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}

	return os.WriteFile(regPath, out, 0o644)
}

// argoCDValuesFile is the path to the ArgoCD app-of-apps values file relative to project root.
const argoCDValuesFile = "k8s/argocd/charts/apps/values.yaml"

// argoCDServiceGroups represents the serviceGroups section of the ArgoCD values file.
type argoCDServiceGroups struct {
	ServiceGroups map[string]*argoCDGroupEntry `yaml:"serviceGroups"`
}

// argoCDGroupEntry represents a single group in the ArgoCD serviceGroups.
type argoCDGroupEntry struct {
	SyncWave  string              `yaml:"syncWave"`
	ChartPath string              `yaml:"chartPath"`
	Services  []argoCDServiceName `yaml:"services"`
}

// argoCDServiceName represents a service entry in an ArgoCD group.
type argoCDServiceName struct {
	Name string `yaml:"name"`
}

// registerArgoCDService registers the service group in the ArgoCD app-of-apps values.yaml.
// If the group already exists, the service is appended to its services list.
// If the group does not exist, it is created with the service as the first entry.
func registerArgoCDService(projectRoot string, cfg *config.ProjectConfig, ctx *engine.TemplateContext) error {
	svc := ctx.Service
	valuesPath := filepath.Join(projectRoot, argoCDValuesFile)

	if !fsutil.FileExists(valuesPath) {
		return fmt.Errorf("ArgoCD values file not found at %s", valuesPath)
	}

	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("read ArgoCD values: %w", err)
	}

	// Parse the full values file into a generic map to preserve all fields
	var fullValues map[string]interface{}
	if err := yaml.Unmarshal(data, &fullValues); err != nil {
		return fmt.Errorf("parse ArgoCD values: %w", err)
	}

	// Extract or initialize the serviceGroups section
	var groups argoCDServiceGroups
	if err := yaml.Unmarshal(data, &groups); err != nil {
		return fmt.Errorf("parse ArgoCD serviceGroups: %w", err)
	}

	if groups.ServiceGroups == nil {
		groups.ServiceGroups = make(map[string]*argoCDGroupEntry)
	}

	groupEntry, exists := groups.ServiceGroups[svc.Group]
	if exists {
		// Check if the service is already registered in this group
		for _, s := range groupEntry.Services {
			if s.Name == svc.Name {
				return nil // Already registered
			}
		}
		// Append service to existing group
		groupEntry.Services = append(groupEntry.Services, argoCDServiceName{Name: svc.Name})
	} else {
		// Create new group entry
		groups.ServiceGroups[svc.Group] = &argoCDGroupEntry{
			SyncWave:  "4",
			ChartPath: svc.Group,
			Services:  []argoCDServiceName{{Name: svc.Name}},
		}
	}

	// Merge the updated serviceGroups back into the full values map
	fullValues["serviceGroups"] = groups.ServiceGroups

	out, err := yaml.Marshal(fullValues)
	if err != nil {
		return fmt.Errorf("marshal ArgoCD values: %w", err)
	}

	return os.WriteFile(valuesPath, out, 0o644)
}

func newGeneratePackageCmd() *cobra.Command {
	var (
		scope    string
		language string
	)

	cmd := &cobra.Command{
		Use:   "package <name>",
		Short: "Generate a new shared package",
		Long: `Generate a new shared package in the monorepo. The package is registered
in ironplate.yaml and workspace configuration files are updated.

Examples:
  iron generate package auth-utils --language node
  iron generate package grpc-client --language go
  iron generate package ui-kit --scope @oss`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := args[0]
			printer := tui.NewStatusPrinter()

			// Find and load project config
			cfgPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("not in an ironplate project: %w", err)
			}

			cfgData, err := os.ReadFile(cfgPath)
			if err != nil {
				return fmt.Errorf("read config: %w", err)
			}

			cfg, err := config.Parse(cfgData)
			if err != nil {
				return fmt.Errorf("parse config: %w", err)
			}

			projectRoot := filepath.Dir(cfgPath)

			// Check for duplicate package
			for _, pkg := range cfg.Spec.Packages {
				if pkg.Name == packageName && pkg.Language == language {
					return fmt.Errorf("package %q (%s) already exists in ironplate.yaml", packageName, language)
				}
			}

			// Default scope
			if scope == "" {
				scope = "@" + cfg.Metadata.Organization
			}

			// Determine template and output directory
			templateDir := "package/" + language
			var packageDir string
			switch language {
			case "node":
				packageDir = filepath.Join(projectRoot, "packages", "node", packageName)
			case "go":
				packageDir = filepath.Join(projectRoot, "packages", "go", packageName)
			default:
				return fmt.Errorf("unsupported language: %s", language)
			}

			// Build template context
			ctx := engine.NewTemplateContext(cfg)
			ctx.Package = &engine.PackageTemplateData{
				Name:       packageName,
				NamePascal: strings.ReplaceAll(strings.Title(strings.ReplaceAll(packageName, "-", " ")), " ", ""), //nolint:staticcheck
				NameSnake:  strings.ReplaceAll(packageName, "-", "_"),
				Language:   language,
				Scope:      scope,
			}

			renderer := engine.NewRenderer()
			totalSteps := 4
			step := 0

			printer.Section("Generating package: " + packageName)

			// 1. Create package files
			step++
			printer.Step(step, totalSteps, "Creating package files")
			if err := renderer.RenderFS(templates.FS, templateDir, packageDir, ctx); err != nil {
				return fmt.Errorf("render package files: %w", err)
			}

			// 2. Update workspace configuration
			step++
			printer.Step(step, totalSteps, "Updating workspace configuration")
			switch language {
			case "go":
				if err := updateGoWorkspace(projectRoot, packageName); err != nil {
					printer.Warning(fmt.Sprintf("Could not update go.work: %s", err))
				}
			}

			// 3. Update ironplate.yaml
			step++
			printer.Step(step, totalSteps, "Updating ironplate.yaml")
			cfg.Spec.Packages = append(cfg.Spec.Packages, config.PackageSpec{
				Name:     packageName,
				Scope:    scope,
				Language: language,
			})

			cfgData, err = yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}
			if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			// 4. Summary
			step++
			printer.Step(step, totalSteps, "Done")
			fmt.Println()
			printer.Success(fmt.Sprintf("Package %s created successfully!", packageName))
			printer.Info(fmt.Sprintf("Path:  packages/%s/%s/", language, packageName))
			printer.Info(fmt.Sprintf("Scope: %s/%s", scope, packageName))
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "", "Package scope (e.g., @myproject)")
	cmd.Flags().StringVar(&language, "language", "node", "Language: node, go")

	return cmd
}

// updateGoWorkspace adds a new package to the go.work use directives.
func updateGoWorkspace(projectRoot, packageName string) error {
	goWorkPath := filepath.Join(projectRoot, "go.work")
	if !fsutil.FileExists(goWorkPath) {
		return nil
	}

	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return err
	}

	content := string(data)
	useEntry := fmt.Sprintf("./packages/go/%s", packageName)

	if strings.Contains(content, useEntry) {
		return nil // Already present
	}

	// Find the closing paren of the use block and insert before it
	useBlock := strings.Index(content, "use (")
	if useBlock == -1 {
		content += fmt.Sprintf("\nuse (\n\t%s\n)\n", useEntry)
		return os.WriteFile(goWorkPath, []byte(content), 0o644)
	}

	closeParen := strings.Index(content[useBlock:], "\n)")
	if closeParen == -1 {
		return fmt.Errorf("malformed go.work: no closing paren for use block")
	}
	insertAt := useBlock + closeParen
	content = content[:insertAt] + "\n\t" + useEntry + content[insertAt:]

	return os.WriteFile(goWorkPath, []byte(content), 0o644)
}
