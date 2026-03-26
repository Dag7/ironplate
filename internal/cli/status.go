package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/internal/tui"
	"github.com/ironplate-dev/ironplate/internal/version"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show project status",
		Long:  `Display the current project configuration, installed components, services, and ironplate version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("no ironplate project found: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			printStatus(cfg, configPath)
			return nil
		},
	}
}

func printStatus(cfg *config.ProjectConfig, configPath string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Println()
	fmt.Println(tui.BoldStyle.Render("  Project Status"))
	fmt.Println()

	fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Project:"), cfg.Metadata.Name)
	fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Organization:"), cfg.Metadata.Organization)
	if cfg.Metadata.Domain != "" {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Domain:"), cfg.Metadata.Domain)
	}
	fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Config:"), configPath)
	w.Flush()

	fmt.Println()

	// Languages
	fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Languages:"), strings.Join(cfg.Spec.Languages, ", "))
	w.Flush()

	// Cloud
	if cfg.Spec.Cloud.Provider != "" && cfg.Spec.Cloud.Provider != "none" {
		provider := cfg.Spec.Cloud.Provider
		if cfg.Spec.Cloud.Region != "" {
			provider += " (" + cfg.Spec.Cloud.Region + ")"
		}
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Cloud:"), provider)

		if len(cfg.Spec.Cloud.Environments) > 0 {
			var envNames []string
			for _, env := range cfg.Spec.Cloud.Environments {
				envNames = append(envNames, env.Name)
			}
			fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Environments:"), strings.Join(envNames, ", "))
		}
		w.Flush()
	}

	// Components
	if len(cfg.Spec.Infrastructure.Components) > 0 {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Components:"), strings.Join(cfg.Spec.Infrastructure.Components, ", "))
	} else {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Components:"), tui.MutedStyle.Render("none"))
	}
	w.Flush()

	// Database
	if cfg.Spec.Infrastructure.Database.Type != "" {
		db := cfg.Spec.Infrastructure.Database.Type
		if cfg.Spec.Infrastructure.Database.Version != "" {
			db += " " + cfg.Spec.Infrastructure.Database.Version
		}
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Database:"), db)
		w.Flush()
	}

	// Services
	fmt.Println()
	if len(cfg.Spec.Services) > 0 {
		fmt.Fprintf(w, "  %s\t%d service(s)\n", tui.BoldStyle.Render("Services:"), len(cfg.Spec.Services))
		w.Flush()
		for _, svc := range cfg.Spec.Services {
			fmt.Fprintf(w, "    %s %s\t%s\tport %d\n", tui.ArrowMark, svc.Name, tui.MutedStyle.Render(svc.Type), svc.Port)
		}
	} else {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Services:"), tui.MutedStyle.Render("none"))
	}
	w.Flush()

	// Dev Environment
	fmt.Println()
	if cfg.Spec.DevEnvironment.Type != "" {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Dev Environment:"), cfg.Spec.DevEnvironment.Type)
	}
	if cfg.Spec.DevEnvironment.K8sLocal != "" {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Local K8s:"), cfg.Spec.DevEnvironment.K8sLocal)
	}
	if cfg.Spec.DevEnvironment.DevTool != "" {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Dev Tool:"), cfg.Spec.DevEnvironment.DevTool)
	}
	w.Flush()

	// CI/CD
	if cfg.Spec.CICD.Platform != "" {
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("CI/CD:"), cfg.Spec.CICD.Platform)
		w.Flush()
	}

	// GitOps
	if cfg.Spec.GitOps.Enabled {
		tool := cfg.Spec.GitOps.Tool
		if tool == "" {
			tool = "enabled"
		}
		fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("GitOps:"), tool)
		w.Flush()
	}

	// Version
	fmt.Println()
	fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Ironplate:"), "v"+version.Short())
	w.Flush()
	fmt.Println()
}
