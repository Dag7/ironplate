package scaffold

import (
	"fmt"
	"strings"

	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/internal/tui"
)

// PrintSummary displays a post-scaffold summary with next steps.
func PrintSummary(cfg *config.ProjectConfig, outputDir string) {
	printer := tui.NewStatusPrinter()

	printer.Section("Project Summary")
	fmt.Printf("  Name:         %s\n", tui.BoldStyle.Render(cfg.Metadata.Name))
	fmt.Printf("  Organization: %s\n", cfg.Metadata.Organization)
	fmt.Printf("  Languages:    %s\n", strings.Join(cfg.Spec.Languages, ", "))
	fmt.Printf("  Cloud:        %s\n", cfg.Spec.Cloud.Provider)

	if len(cfg.Spec.Infrastructure.Components) > 0 {
		fmt.Printf("  Components:   %s\n", strings.Join(cfg.Spec.Infrastructure.Components, ", "))
	}

	if len(cfg.Spec.Services) > 0 {
		fmt.Printf("  Services:     %d\n", len(cfg.Spec.Services))
	}

	printer.Section("Next Steps")
	printer.Info(fmt.Sprintf("cd %s", outputDir))

	if cfg.Spec.DevEnvironment.Type == "devcontainer" {
		printer.Info("Open in VS Code and reopen in container")
	}

	if cfg.Spec.HasLanguage("node") {
		switch cfg.Spec.Monorepo.PackageManager {
		case "yarn":
			printer.Info("yarn install")
		case "pnpm":
			printer.Info("pnpm install")
		}
	}

	if cfg.Spec.DevEnvironment.DevTool == "tilt" {
		printer.Info("tilt up  # Start local development")
	}

	fmt.Println()
	printer.Info("iron generate service <name>  # Add a new service")
	printer.Info("iron add <component>          # Add infrastructure")
	printer.Info("iron doctor                   # Check prerequisites")
	fmt.Println()
}
