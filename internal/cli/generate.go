package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/engine"
	"github.com/dag7/ironplate/internal/scaffold"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/dag7/ironplate/pkg/fsutil"
	"github.com/dag7/ironplate/templates"
)

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

			pc, err := loadProject()
			if err != nil {
				return err
			}
			cfg := pc.Config

			// Check for duplicate service name
			for _, svc := range cfg.Spec.Services {
				if svc.Name == serviceName {
					return fmt.Errorf("service %q already exists in ironplate.yaml", serviceName)
				}
			}

			// Validate service type
			if _, ok := config.ServiceTemplateDirs[serviceType]; !ok {
				supported := make([]string, 0, len(config.ServiceTemplateDirs))
				for k := range config.ServiceTemplateDirs {
					supported = append(supported, k)
				}
				return fmt.Errorf("unsupported service type %q — supported: %s", serviceType, strings.Join(supported, ", "))
			}

			if group == "" {
				group = "core"
			}
			if port == 0 {
				port = scaffold.NextForwardPort(cfg.Spec.Services)
			}
			debugPort := scaffold.NextDebugForwardPort(cfg.Spec.Services, serviceType)

			var featureList []string
			if features != "" {
				featureList = strings.Split(features, ",")
			}

			ctx := engine.NewTemplateContext(cfg)
			ctx.Service = &engine.ServiceTemplateData{
				Name:      serviceName,
				Type:      serviceType,
				Group:     group,
				Port:      port,
				DebugPort: debugPort,
				SrcFolder: "apps",
				Features:  featureList,
			}

			printer.Section("Generating service: " + serviceName)

			// Use the shared service rendering pipeline
			warnings, renderErr := scaffold.RenderService(scaffold.RenderServiceParams{
				Renderer:    engine.NewRenderer(),
				Templates:   templates.FS,
				ProjectRoot: pc.ProjectRoot,
				Config:      cfg,
				Ctx:         ctx,
			})
			if renderErr != nil {
				return renderErr
			}
			for _, w := range warnings {
				printer.Warning(w)
			}

			// Update ironplate.yaml
			cfg.Spec.Services = append(cfg.Spec.Services, config.ServiceSpec{
				Name:     serviceName,
				Type:     serviceType,
				Group:    group,
				Port:     port,
				Features: featureList,
			})
			if err := pc.saveConfig(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

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

			// Load project config
			pc, err := loadProject()
			if err != nil {
				return err
			}
			cfg := pc.Config
			projectRoot := pc.ProjectRoot

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

			if err := pc.saveConfig(); err != nil {
				return fmt.Errorf("save config: %w", err)
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
