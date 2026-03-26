// Package cli defines the root command and global flags for the iron CLI.
package cli

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	quiet   bool
	noColor bool
	cfgFile string
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iron",
		Short: "Scaffold production-grade Kubernetes development environments",
		Long: `Ironplate (iron) generates complete, production-ready project scaffolding
including devcontainers, k3d clusters, Tilt-based development, Helm charts,
Pulumi IaC, ArgoCD GitOps, CI/CD pipelines, and multi-language service templates.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			configureLogging()
		},
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default: ./ironplate.yaml)")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newTiltCmd())
	cmd.AddCommand(newDevCmd())

	return cmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

func configureLogging() {
	switch {
	case verbose:
		log.SetLevel(log.DebugLevel)
	case quiet:
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}
