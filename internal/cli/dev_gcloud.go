package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newDevGCloudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcloud",
		Short: "GCP authentication helpers",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "login",
		Short: "Full GCP authentication (browser + ADC)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gcloudLogin()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "adc",
		Short: "Application Default Credentials only",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gcloudADC()
		},
	})

	return cmd
}

func gcloudLogin() error {
	printer := tui.NewStatusPrinter()

	printer.Info("Step 1/2: GCP authentication...")
	loginCmd := exec.Command("gcloud", "auth", "login")
	loginCmd.Stdin = os.Stdin
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("gcloud auth login failed: %w", err)
	}
	printer.Success("GCP authentication complete")

	printer.Info("Step 2/2: Application Default Credentials...")
	adcCmd := exec.Command("gcloud", "auth", "application-default", "login")
	adcCmd.Stdin = os.Stdin
	adcCmd.Stdout = os.Stdout
	adcCmd.Stderr = os.Stderr
	if err := adcCmd.Run(); err != nil {
		return fmt.Errorf("gcloud ADC login failed: %w", err)
	}
	printer.Success("ADC authentication complete")
	return nil
}

func gcloudADC() error {
	printer := tui.NewStatusPrinter()
	printer.Info("Setting up Application Default Credentials...")
	cmd := exec.Command("gcloud", "auth", "application-default", "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gcloud ADC login failed: %w", err)
	}
	printer.Success("ADC authentication complete")
	return nil
}

func runGCloudInteractive() error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("GCloud").
		Options(
			huh.NewOption("Full login (browser + ADC)", "login"),
			huh.NewOption("Application Default Credentials only", "adc"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "login":
		return gcloudLogin()
	case "adc":
		return gcloudADC()
	}
	return nil
}
