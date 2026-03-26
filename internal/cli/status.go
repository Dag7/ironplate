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

			return printStatus(cfg, configPath)
		},
	}
}

func printStatus(cfg *config.ProjectConfig, configPath string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Println()
	fmt.Println(tui.BoldStyle.Render("  Project Status"))
	fmt.Println()

	_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Project:"), cfg.Metadata.Name)
	_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Organization:"), cfg.Metadata.Organization)
	if cfg.Metadata.Domain != "" {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Domain:"), cfg.Metadata.Domain)
	}
	_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Config:"), configPath)
	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Println()

	// Languages
	_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Languages:"), strings.Join(cfg.Spec.Languages, ", "))
	if err := w.Flush(); err != nil {
		return err
	}

	// Cloud
	if cfg.Spec.Cloud.Provider != "" && cfg.Spec.Cloud.Provider != "none" {
		provider := cfg.Spec.Cloud.Provider
		if cfg.Spec.Cloud.Region != "" {
			provider += " (" + cfg.Spec.Cloud.Region + ")"
		}
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Cloud:"), provider)

		if len(cfg.Spec.Cloud.Environments) > 0 {
			var envNames []string
			for _, env := range cfg.Spec.Cloud.Environments {
				envNames = append(envNames, env.Name)
			}
			_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Environments:"), strings.Join(envNames, ", "))
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	// Components
	if len(cfg.Spec.Infrastructure.Components) > 0 {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Components:"), strings.Join(cfg.Spec.Infrastructure.Components, ", "))
	} else {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Components:"), tui.MutedStyle.Render("none"))
	}
	if err := w.Flush(); err != nil {
		return err
	}

	// Database
	if cfg.Spec.Infrastructure.Database.Type != "" {
		db := cfg.Spec.Infrastructure.Database.Type
		if cfg.Spec.Infrastructure.Database.Version != "" {
			db += " " + cfg.Spec.Infrastructure.Database.Version
		}
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Database:"), db)
		if err := w.Flush(); err != nil {
			return err
		}
	}

	// Services
	fmt.Println()
	if len(cfg.Spec.Services) > 0 {
		_, _ = fmt.Fprintf(w, "  %s\t%d service(s)\n", tui.BoldStyle.Render("Services:"), len(cfg.Spec.Services))
		if err := w.Flush(); err != nil {
			return err
		}
		for _, svc := range cfg.Spec.Services {
			_, _ = fmt.Fprintf(w, "    %s %s\t%s\tport %d\n", tui.ArrowMark, svc.Name, tui.MutedStyle.Render(svc.Type), svc.Port)
		}
	} else {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Services:"), tui.MutedStyle.Render("none"))
	}
	if err := w.Flush(); err != nil {
		return err
	}

	// Dev Environment
	fmt.Println()
	if cfg.Spec.DevEnvironment.Type != "" {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Dev Environment:"), cfg.Spec.DevEnvironment.Type)
	}
	if cfg.Spec.DevEnvironment.K8sLocal != "" {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Local K8s:"), cfg.Spec.DevEnvironment.K8sLocal)
	}
	if cfg.Spec.DevEnvironment.DevTool != "" {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Dev Tool:"), cfg.Spec.DevEnvironment.DevTool)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	// CI/CD
	if cfg.Spec.CICD.Platform != "" {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("CI/CD:"), cfg.Spec.CICD.Platform)
		if err := w.Flush(); err != nil {
			return err
		}
	}

	// GitOps
	if cfg.Spec.GitOps.Enabled {
		tool := cfg.Spec.GitOps.Tool
		if tool == "" {
			tool = "enabled"
		}
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("GitOps:"), tool)
		if err := w.Flush(); err != nil {
			return err
		}
	}

	// Version
	fmt.Println()
	_, _ = fmt.Fprintf(w, "  %s\t%s\n", tui.BoldStyle.Render("Ironplate:"), "v"+version.Short())
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Println()

	return nil
}
