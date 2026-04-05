package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/scaffold"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/dag7/ironplate/templates"
)

func newInitCmd() *cobra.Command {
	var (
		name            string
		organization    string
		domain          string
		language        string
		provider        string
		preset          string
		tools           string
		nonInteractive  bool
		exampleServices bool
	)

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new ironplate project",
		Long: `Interactively scaffold a new production-grade Kubernetes development environment.

Uses an interactive TUI to guide you through project configuration.
Use --non-interactive with flags for scripted usage.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()

			// Show banner
			tui.PrintBanner()

			var cfg *config.ProjectConfig
			wantExampleServices := exampleServices

			if nonInteractive {
				// Validate required flags
				if name == "" {
					return fmt.Errorf("--name is required in non-interactive mode")
				}
				if organization == "" {
					return fmt.Errorf("--organization is required in non-interactive mode")
				}
				if domain == "" {
					return fmt.Errorf("--domain is required in non-interactive mode")
				}

				cfg = config.NewDefaultConfig(name, organization, domain)
				applyFlags(cfg, language, provider, preset, tools)
			} else {
				// Interactive TUI prompts
				var err error
				cfg, wantExampleServices, err = runInteractivePrompts(name, organization, domain, language, provider, preset)
				if err != nil {
					return err
				}
			}

			// Determine output directory
			outputDir := cfg.Metadata.Name
			if len(args) > 0 {
				outputDir = args[0]
			}
			outputDir, err := filepath.Abs(outputDir)
			if err != nil {
				return fmt.Errorf("resolve output path: %w", err)
			}

			// Check if directory already exists
			if _, err := os.Stat(outputDir); err == nil {
				entries, readErr := os.ReadDir(outputDir)
				if readErr != nil {
					return fmt.Errorf("read output directory %s: %w", outputDir, readErr)
				}
				if len(entries) > 0 {
					return fmt.Errorf("directory %s already exists and is not empty", outputDir)
				}
			}

			// Validate
			result := scaffold.ValidateForScaffold(cfg, outputDir)
			if !result.IsValid() {
				for _, e := range result.Errors {
					printer.Error(e)
				}
				return fmt.Errorf("validation failed with %d error(s)", len(result.Errors))
			}
			for _, w := range result.Warnings {
				printer.Warning(w)
			}

			fmt.Println()
			printer.Section("Configuration Summary")
			fmt.Printf("  Project:    %s\n", cfg.Metadata.Name)
			fmt.Printf("  Org:        %s\n", cfg.Metadata.Organization)
			fmt.Printf("  Domain:     %s\n", cfg.Metadata.Domain)
			fmt.Printf("  Languages:  %v\n", cfg.Spec.Languages)
			fmt.Printf("  Provider:   %s\n", cfg.Spec.Cloud.Provider)
			fmt.Printf("  Components: %v\n", cfg.Spec.Infrastructure.Components)
			fmt.Printf("  Output:     %s\n", outputDir)
			fmt.Println()

			// Scaffold the project
			scaffolder := scaffold.NewScaffolder(cfg, outputDir, templates.FS)
			if err := scaffolder.Scaffold(); err != nil {
				return fmt.Errorf("scaffold failed: %w", err)
			}

			// Generate example services if requested
			if wantExampleServices {
				services := scaffold.DefaultExampleServices(cfg)
				if len(services) > 0 {
					fmt.Println()
					if err := scaffold.GenerateExampleServices(cfg, outputDir, templates.FS, services); err != nil {
						printer.Warning(fmt.Sprintf("Could not generate example services: %s", err))
					}
				}
			}

			// Print summary
			scaffold.PrintSummary(cfg, outputDir)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Project name (kebab-case)")
	cmd.Flags().StringVar(&organization, "org", "", "Organization name")
	cmd.Flags().StringVar(&domain, "domain", "", "Base domain (e.g., myplatform.dev)")
	cmd.Flags().StringVar(&language, "language", "", "Primary language: node, go, mixed")
	cmd.Flags().StringVar(&provider, "provider", "", "Cloud provider: gcp, aws, azure, none")
	cmd.Flags().StringVar(&preset, "preset", "", "Component preset: minimal, standard, full")
	cmd.Flags().StringVar(&tools, "tools", "", "Dev tools to install (comma-separated): operator-sdk,git-secret,mc,kompose or 'all'")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Skip interactive prompts")
	cmd.Flags().BoolVar(&exampleServices, "example-services", false, "Generate example frontend and backend services")

	return cmd
}

func runInteractivePrompts(name, org, domain, language, provider, preset string) (*config.ProjectConfig, bool, error) {
	// Project Name
	if name == "" {
		err := huh.NewInput().
			Title("Project name").
			Description("Kebab-case name for your project (e.g., my-platform)").
			Placeholder("my-platform").
			Value(&name).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("project name is required")
				}
				return nil
			}).
			Run()
		if err != nil {
			return nil, false, err
		}
	}

	// Organization
	if org == "" {
		defaultOrg := name
		org = defaultOrg
		err := huh.NewInput().
			Title("Organization").
			Description("npm scope (@org/pkg) and Go module path (github.com/org/project)").
			Placeholder(defaultOrg).
			Value(&org).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("organization is required")
				}
				return nil
			}).
			Run()
		if err != nil {
			return nil, false, err
		}
	}

	// Domain
	if domain == "" {
		err := huh.NewInput().
			Title("Domain").
			Description("Base domain for your project (e.g., myplatform.dev)").
			Placeholder("myplatform.dev").
			Value(&domain).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("domain is required")
				}
				return nil
			}).
			Run()
		if err != nil {
			return nil, false, err
		}
	}

	// Language selection
	if language == "" {
		err := huh.NewSelect[string]().
			Title("Languages").
			Description("Which languages will your services use?").
			Options(
				huh.NewOption("Node.js (TypeScript)", "node"),
				huh.NewOption("Go", "go"),
				huh.NewOption("Both (Node.js + Go)", "mixed"),
			).
			Value(&language).
			Run()
		if err != nil {
			return nil, false, err
		}
	}

	// Cloud provider
	if provider == "" {
		err := huh.NewSelect[string]().
			Title("Cloud Provider").
			Description("Target cloud platform for infrastructure").
			Options(
				huh.NewOption("Google Cloud Platform (GCP)", "gcp"),
				huh.NewOption("Amazon Web Services (AWS)", "aws"),
				huh.NewOption("Microsoft Azure", "azure"),
				huh.NewOption("None (local only)", "none"),
			).
			Value(&provider).
			Run()
		if err != nil {
			return nil, false, err
		}
	}

	// Infrastructure preset
	if preset == "" {
		err := huh.NewSelect[string]().
			Title("Infrastructure Preset").
			Description("Choose a set of infrastructure components").
			Options(
				huh.NewOption("Minimal (no infra components)", "minimal"),
				huh.NewOption("Standard (Redis, Kafka, Hasura, External Secrets)", "standard"),
				huh.NewOption("Full (all components + GitOps + Observability)", "full"),
			).
			Value(&preset).
			Run()
		if err != nil {
			return nil, false, err
		}
	}

	// Dev tools (optional)
	var selectedTools []string
	toolOptions := make([]huh.Option[string], 0, len(config.AvailableDevTools))
	for _, t := range config.AvailableDevTools {
		toolOptions = append(toolOptions, huh.NewOption(t.Description+" ("+t.Name+")", t.Name))
	}

	if err := huh.NewMultiSelect[string]().
		Title("Additional Dev Tools").
		Description("Select optional tools for the dev container (space to toggle)").
		Options(toolOptions...).
		Value(&selectedTools).
		Run(); err != nil {
		return nil, false, err
	}

	// Example services
	wantExamples := true
	if err := huh.NewConfirm().
		Title("Generate example services?").
		Description("Creates a starter backend API and frontend app so you can see the full e2e flow immediately").
		Affirmative("Yes").
		Negative("No").
		Value(&wantExamples).
		Run(); err != nil {
		return nil, false, err
	}

	cfg := config.NewDefaultConfig(name, org, domain)
	applyFlags(cfg, language, provider, preset, "")
	cfg.Spec.DevEnvironment.Tools = selectedTools

	return cfg, wantExamples, nil
}

func applyFlags(cfg *config.ProjectConfig, language, provider, preset, tools string) {
	applyLanguage(cfg, language)
	applyProvider(cfg, provider)
	applyPreset(cfg, preset)
	applyTools(cfg, tools)
	syncGitOpsFlag(cfg)
}

var languageMap = map[string][]string{
	"node":  {"node"},
	"go":    {"go"},
	"mixed": {"node", "go"},
}

func applyLanguage(cfg *config.ProjectConfig, language string) {
	if langs, ok := languageMap[language]; ok {
		cfg.Spec.Languages = langs
	}
}

func applyProvider(cfg *config.ProjectConfig, provider string) {
	if provider == "" {
		return
	}
	cfg.Spec.Cloud.Provider = provider
	if provider == "none" {
		cfg.Spec.Cloud.Environments = nil
		cfg.Spec.GitOps.Enabled = false
	}
}

func applyPreset(cfg *config.ProjectConfig, preset string) {
	if comps, ok := config.Presets[preset]; ok {
		cfg.Spec.Infrastructure.Components = comps
	}
}

func applyTools(cfg *config.ProjectConfig, tools string) {
	switch {
	case tools == "all":
		allTools := make([]string, 0, len(config.AvailableDevTools))
		for _, t := range config.AvailableDevTools {
			allTools = append(allTools, t.Name)
		}
		cfg.Spec.DevEnvironment.Tools = allTools
	case tools != "":
		cfg.Spec.DevEnvironment.Tools = strings.Split(tools, ",")
	}
}

func syncGitOpsFlag(cfg *config.ProjectConfig) {
	if !cfg.Spec.Infrastructure.HasComponent("argocd") {
		cfg.Spec.GitOps.Enabled = false
	}
}
