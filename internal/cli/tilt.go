package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/tiltmgr"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newTiltCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tilt",
		Aliases: []string{"t"},
		Short:   "Manage Tilt development environment",
		Long:    `Launch, stop, and manage Tilt profiles for local Kubernetes development.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTiltInteractive()
		},
	}

	cmd.AddCommand(newTiltUpCmd())
	cmd.AddCommand(newTiltDownCmd())
	cmd.AddCommand(newTiltStatusCmd())
	cmd.AddCommand(newTiltEnableCmd())
	cmd.AddCommand(newTiltRetryCmd())
	cmd.AddCommand(newTiltProfileCmd())
	cmd.AddCommand(newTiltServiceCmd())

	return cmd
}

// --- iron tilt up ---

func newTiltUpCmd() *cobra.Command {
	var (
		force     bool
		noBrowser bool
		all       bool
		add       string
	)

	cmd := &cobra.Command{
		Use:   "up [profile]",
		Short: "Start Tilt with a profile",
		Long:  `Start Tilt using the specified profile. Defaults to "default" if no profile is given.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}

			printer := tui.NewStatusPrinter()
			pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))

			// Check if Tilt is already running
			if !force && tiltmgr.IsRunning() {
				// If --add is specified, enable additional services
				if add != "" {
					services := strings.Split(add, ",")
					printer.Info(fmt.Sprintf("Enabling %d additional resources...", len(services)))
					return tiltmgr.Enable(services)
				}
				printer.Warning("Tilt is already running. Use --force to restart or --add to add services.")
				return nil
			}

			// --all: start everything
			if all {
				printer.Info("Starting all Tilt resources...")
				if err := chdir(projectRoot); err != nil {
					return err
				}
				return tiltmgr.Up(&tiltmgr.Profile{}, noBrowser)
			}

			// Load profile
			profileName := "default"
			if len(args) > 0 {
				profileName = args[0]
			}

			profile, err := pm.Load(profileName)
			if err != nil {
				return fmt.Errorf("profile %q: %w", profileName, err)
			}

			totalResources := len(profile.Services) + len(profile.Infra)
			printer.Info(fmt.Sprintf("Starting Tilt with profile %q (%d resources)", profileName, totalResources))

			if err := chdir(projectRoot); err != nil {
				return err
			}
			return tiltmgr.Up(profile, noBrowser)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force restart if Tilt is already running")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open the Tilt browser UI")
	cmd.Flags().BoolVar(&all, "all", false, "Start all resources (no profile filter)")
	cmd.Flags().StringVar(&add, "add", "", "Add services to running Tilt (comma-separated)")

	return cmd
}

// --- iron tilt down ---

func newTiltDownCmd() *cobra.Command {
	var (
		all      bool
		services string
	)

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop Tilt or specific resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()
			if !tiltmgr.IsRunning() {
				printer.Info("Tilt is not running.")
				return nil
			}

			// Specific services to disable
			if services != "" {
				svcList := strings.Split(services, ",")
				printer.Info(fmt.Sprintf("Disabling %d resources...", len(svcList)))
				return tiltmgr.Disable(svcList)
			}

			// Default: stop everything
			if all || !cmd.Flags().Changed("services") {
				return tiltmgr.Down()
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Stop everything (default)")
	cmd.Flags().StringVarP(&services, "services", "s", "", "Disable specific services (comma-separated)")

	return cmd
}

// --- iron tilt status ---

func newTiltStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of running Tilt resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printTiltStatus()
		},
	}
}

// --- iron tilt enable ---

func newTiltEnableCmd() *cobra.Command {
	var (
		all      bool
		services string
	)

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable disabled Tilt resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()
			if !tiltmgr.IsRunning() {
				printer.Info("Tilt is not running.")
				return nil
			}

			if services != "" {
				svcList := strings.Split(services, ",")
				printer.Info(fmt.Sprintf("Enabling %d resources...", len(svcList)))
				return tiltmgr.Enable(svcList)
			}

			disabled, err := tiltmgr.GetDisabledResources()
			if err != nil {
				return err
			}
			if len(disabled) == 0 {
				printer.Info("No disabled resources.")
				return nil
			}

			if all {
				printer.Info(fmt.Sprintf("Enabling %d resources...", len(disabled)))
				return tiltmgr.Enable(disabled)
			}

			// Interactive selection
			return runTiltEnableInteractive()
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Enable all disabled resources")
	cmd.Flags().StringVarP(&services, "services", "s", "", "Enable specific services (comma-separated)")

	return cmd
}

// --- iron tilt retry ---

func newTiltRetryCmd() *cobra.Command {
	var (
		all      bool
		resource string
	)

	cmd := &cobra.Command{
		Use:   "retry",
		Short: "Retry errored Tilt resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()
			if !tiltmgr.IsRunning() {
				printer.Info("Tilt is not running.")
				return nil
			}

			if resource != "" {
				printer.Info("Retrying " + resource + "...")
				return tiltmgr.Retry([]string{resource})
			}

			errored, err := tiltmgr.GetErroredResources()
			if err != nil {
				return err
			}
			if len(errored) == 0 {
				printer.Info("No errored resources.")
				return nil
			}

			if all {
				printer.Info(fmt.Sprintf("Retrying %d resources...", len(errored)))
				return tiltmgr.Retry(errored)
			}

			return runTiltRetryInteractive()
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Retry all errored resources")
	cmd.Flags().StringVarP(&resource, "resource", "r", "", "Retry a specific resource")

	return cmd
}

// --- iron tilt profile ---

func newTiltProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage Tilt launch profiles",
	}

	cmd.AddCommand(newTiltProfileListCmd())
	cmd.AddCommand(newTiltProfileShowCmd())
	cmd.AddCommand(newTiltProfileCreateCmd())
	cmd.AddCommand(newTiltProfileEditCmd())
	cmd.AddCommand(newTiltProfileDeleteCmd())

	return cmd
}

func newTiltProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}
			pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))
			return printProfileList(pm)
		},
	}
}

func newTiltProfileShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show profile details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}
			pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))

			if len(args) == 0 {
				return showProfileInteractive(pm)
			}

			profile, err := pm.Load(args[0])
			if err != nil {
				return err
			}
			printProfileBox(profile)
			return nil
		},
	}
}

func newTiltProfileCreateCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}

			pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))

			if pm.Exists(name) {
				return fmt.Errorf("profile %q already exists", name)
			}

			tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
			discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
			if err != nil {
				return fmt.Errorf("parse Tiltfile: %w", err)
			}

			profile, err := interactiveProfileSelect(name, description, discovered, nil)
			if err != nil {
				return err
			}

			if err := pm.Save(profile); err != nil {
				return err
			}

			printer := tui.NewStatusPrinter()
			printer.Success(fmt.Sprintf("Profile %q created", name))
			printProfileBox(profile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Profile description")
	return cmd
}

func newTiltProfileEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit an existing profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}

			pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))

			existing, err := pm.Load(name)
			if err != nil {
				return err
			}

			tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
			discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
			if err != nil {
				return fmt.Errorf("parse Tiltfile: %w", err)
			}

			profile, err := interactiveProfileSelect(name, existing.Description, discovered, existing)
			if err != nil {
				return err
			}

			if err := pm.Save(profile); err != nil {
				return err
			}

			printer := tui.NewStatusPrinter()
			printer.Success(fmt.Sprintf("Profile %q updated", name))
			printProfileBox(profile)
			return nil
		},
	}
}

func newTiltProfileDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}

			pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))

			if !yes {
				var confirm bool
				if err := huh.NewConfirm().
					Title(fmt.Sprintf("Delete profile %q?", name)).
					Value(&confirm).
					Run(); err != nil {
					return err
				}
				if !confirm {
					return nil
				}
			}

			if err := pm.Delete(name); err != nil {
				return err
			}

			tui.NewStatusPrinter().Success(fmt.Sprintf("Profile %q deleted", name))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

// --- iron tilt service ---

func newTiltServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "List available Tilt services and groups",
	}

	cmd.AddCommand(newTiltServiceListCmd())
	cmd.AddCommand(newTiltServiceGroupsCmd())

	return cmd
}

func newTiltServiceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all discoverable services and infra",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}
			tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
			discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
			if err != nil {
				return fmt.Errorf("parse Tiltfile: %w", err)
			}
			return printServiceList(discovered)
		},
	}
}

func newTiltServiceGroupsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "groups",
		Short: "List services grouped by category",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, _, err := findProject()
			if err != nil {
				return err
			}
			tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
			discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
			if err != nil {
				return fmt.Errorf("parse Tiltfile: %w", err)
			}
			return printServiceGroups(discovered)
		},
	}
}

// --- helpers ---

// findProject is a convenience wrapper over loadProject for commands that
// only need projectRoot and config (without needing to save config back).
func findProject() (string, *config.ProjectConfig, error) {
	pc, err := loadProject()
	if err != nil {
		return "", nil, err
	}
	return pc.ProjectRoot, pc.Config, nil
}

func chdir(dir string) error {
	return os.Chdir(dir)
}
