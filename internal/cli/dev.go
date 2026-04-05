package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Developer tools for cluster management",
		Long:  `Interactive developer CLI for Kubernetes context, pods, database, and ArgoCD management.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDevInteractive()
		},
	}

	cmd.AddCommand(newDevContextCmd())
	cmd.AddCommand(newDevPodsCmd())
	cmd.AddCommand(newDevDBCmd())
	cmd.AddCommand(newDevArgoCmd())
	cmd.AddCommand(newDevImagesCmd())
	cmd.AddCommand(newDevGCloudCmd())

	return cmd
}

// --- Interactive Menu ---

func runDevInteractive() error {
	_, cfg, err := findProject()
	if err != nil {
		return err
	}

	bannerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(tui.ColorPrimary).
		Padding(0, 2)
	fmt.Printf("\n  %s\n\n", bannerStyle.Render(cfg.Metadata.Name+" dev"))

	for {
		var choice string
		if err := huh.NewSelect[string]().
			Title("What would you like to do?").
			Options(
				huh.NewOption("Context  - Switch Kubernetes context", "context"),
				huh.NewOption("Pods     - Pod management & debugging", "pods"),
				huh.NewOption("Database - Database connections", "db"),
				huh.NewOption("ArgoCD   - GitOps management", "argocd"),
				huh.NewOption("Images   - Container image inspection", "images"),
				huh.NewOption("GCloud   - GCP authentication", "gcloud"),
				huh.NewOption("Exit", "exit"),
			).
			Value(&choice).
			Run(); err != nil {
			return nil // User cancelled
		}

		printer := tui.NewStatusPrinter()
		switch choice {
		case "context":
			if err := runContextInteractive(cfg.Metadata.Name); err != nil {
				printer.Error(err.Error())
			}
		case "pods":
			if err := runPodsInteractive(cfg.Metadata.Name); err != nil {
				printer.Error(err.Error())
			}
		case "db":
			if err := runDBInteractive(cfg.Metadata.Name); err != nil {
				printer.Error(err.Error())
			}
		case "argocd":
			if err := runArgoInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "images":
			if err := runImagesInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "gcloud":
			if err := runGCloudInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "exit":
			return nil
		}
		fmt.Println()
	}
}
