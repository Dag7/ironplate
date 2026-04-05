package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newDevImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "images",
		Aliases: []string{"img"},
		Short:   "Container image inspection",
	}

	cmd.AddCommand(newDevImagesListCmd())
	cmd.AddCommand(newDevImagesLatestCmd())

	return cmd
}

func newDevImagesListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list [service]",
		Short: "List container images for a service",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}

			service := ""
			if len(args) > 0 {
				service = args[0]
			} else {
				if err := huh.NewInput().
					Title("Service name").
					Value(&service).
					Run(); err != nil {
					return nil
				}
			}
			if service == "" {
				return fmt.Errorf("service name required")
			}

			return listContainerImages(cfg, service, limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of images to show")
	return cmd
}

func newDevImagesLatestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "latest [service]",
		Short: "Get latest image tag for a service",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}

			service := ""
			if len(args) > 0 {
				service = args[0]
			} else {
				if err := huh.NewInput().
					Title("Service name").
					Value(&service).
					Run(); err != nil {
					return nil
				}
			}
			if service == "" {
				return fmt.Errorf("service name required")
			}

			return showLatestImage(cfg, service)
		},
	}
}

func listContainerImages(cfg *config.ProjectConfig, service string, limit int) error {
	printer := tui.NewStatusPrinter()

	if cfg.Spec.Cloud.Provider != "gcp" {
		printer.Info("Image listing currently supported for GCP Artifact Registry only.")
		return nil
	}

	registryURL := cfg.Spec.CICD.Registry.URL
	if registryURL == "" {
		return fmt.Errorf("no registry URL configured in ironplate.yaml")
	}

	// Use gcloud to list images
	imagePath := fmt.Sprintf("%s/%s/%s", registryURL, cfg.Metadata.Organization, service)
	cmd := exec.Command("gcloud", "artifacts", "docker", "images", "list", imagePath,
		"--include-tags", "--sort-by=~UPDATE_TIME",
		fmt.Sprintf("--limit=%d", limit), "--format=table(package,tags,createTime)")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func showLatestImage(cfg *config.ProjectConfig, service string) error {
	printer := tui.NewStatusPrinter()

	if cfg.Spec.Cloud.Provider != "gcp" {
		printer.Info("Image inspection currently supported for GCP Artifact Registry only.")
		return nil
	}

	registryURL := cfg.Spec.CICD.Registry.URL
	if registryURL == "" {
		return fmt.Errorf("no registry URL configured in ironplate.yaml")
	}

	imagePath := fmt.Sprintf("%s/%s/%s", registryURL, cfg.Metadata.Organization, service)
	cmd := exec.Command("gcloud", "artifacts", "docker", "images", "list", imagePath,
		"--include-tags", "--sort-by=~UPDATE_TIME",
		"--limit=1", "--format=json")

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query images: %w", err)
	}

	if len(out) > 0 {
		fmt.Printf("\n  %s\n", tui.BoldStyle.Render("Latest image for "+service))
		fmt.Printf("  %s\n", string(out))
	} else {
		printer.Info("No images found for " + service)
	}
	return nil
}

func runImagesInteractive() error {
	_, cfg, err := findProject()
	if err != nil {
		return err
	}

	var choice string
	if err := huh.NewSelect[string]().
		Title("Images").
		Options(
			huh.NewOption("List images for service", "list"),
			huh.NewOption("Show latest image", "latest"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "list":
		var service string
		if err := huh.NewInput().
			Title("Service name").
			Value(&service).
			Run(); err != nil {
			return nil
		}
		if service != "" {
			return listContainerImages(cfg, service, 10)
		}
	case "latest":
		var service string
		if err := huh.NewInput().
			Title("Service name").
			Value(&service).
			Run(); err != nil {
			return nil
		}
		if service != "" {
			return showLatestImage(cfg, service)
		}
	}
	return nil
}
