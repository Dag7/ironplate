package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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

// --- Interactive Menu (no-args mode) ---

func runTiltInteractive() error {
	_, cfg, err := findProject()
	if err != nil {
		return err
	}

	bannerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(tui.ColorPrimary).
		Padding(0, 2)
	fmt.Printf("\n  %s\n\n", bannerStyle.Render(cfg.Metadata.Name+" tilt"))

	for {
		var choice string
		options := []huh.Option[string]{
			huh.NewOption("Up       - Start Tilt with a profile", "up"),
			huh.NewOption("Down     - Stop Tilt", "down"),
			huh.NewOption("Enable   - Enable disabled resources", "enable"),
			huh.NewOption("Retry    - Retry errored resources", "retry"),
			huh.NewOption("Status   - Show resource status", "status"),
			huh.NewOption("Profile  - Manage profiles", "profile"),
			huh.NewOption("Service  - Discover services", "service"),
			huh.NewOption("Exit", "exit"),
		}

		if err := huh.NewSelect[string]().
			Title("What would you like to do?").
			Options(options...).
			Value(&choice).
			Run(); err != nil {
			return nil
		}

		printer := tui.NewStatusPrinter()
		switch choice {
		case "up":
			if err := runTiltUpInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "down":
			if err := runTiltDownInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "enable":
			if err := runTiltEnableInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "retry":
			if err := runTiltRetryInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "status":
			if err := printTiltStatus(); err != nil {
				printer.Error(err.Error())
			}
		case "profile":
			if err := runTiltProfileInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "service":
			if err := runTiltServiceInteractive(); err != nil {
				printer.Error(err.Error())
			}
		case "exit":
			return nil
		}
		fmt.Println()
	}
}

func runTiltUpInteractive() error {
	projectRoot, _, err := findProject()
	if err != nil {
		return err
	}

	pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))
	profiles, err := pm.List()
	if err != nil {
		return err
	}

	options := []huh.Option[string]{
		huh.NewOption("Start all resources", "--all"),
		huh.NewOption("Select custom services", "--custom"),
	}
	for _, p := range profiles {
		label := p.Name
		if p.Description != "" {
			label = fmt.Sprintf("%s - %s", p.Name, p.Description)
		}
		options = append(options, huh.NewOption(label, p.Name))
	}

	var choice string
	if err := huh.NewSelect[string]().
		Title("Start Tilt with").
		Options(options...).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	printer := tui.NewStatusPrinter()

	switch choice {
	case "--all":
		printer.Info("Starting all Tilt resources...")
		return tiltmgr.Up(&tiltmgr.Profile{}, false)
	case "--custom":
		tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
		discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
		if err != nil {
			return err
		}
		profile, err := interactiveProfileSelect("custom", "", discovered, nil)
		if err != nil {
			return err
		}
		printer.Info(fmt.Sprintf("Starting %d resources...", len(profile.Services)+len(profile.Infra)))
		return tiltmgr.Up(profile, false)
	default:
		profile, err := pm.Load(choice)
		if err != nil {
			return err
		}
		printer.Info(fmt.Sprintf("Starting profile %q (%d resources)...", choice, len(profile.Services)+len(profile.Infra)))
		return tiltmgr.Up(profile, false)
	}
}

func runTiltDownInteractive() error {
	printer := tui.NewStatusPrinter()
	if !tiltmgr.IsRunning() {
		printer.Info("Tilt is not running.")
		return nil
	}

	var choice string
	if err := huh.NewSelect[string]().
		Title("Stop Tilt").
		Options(
			huh.NewOption("Stop everything", "all"),
			huh.NewOption("Disable specific resources", "select"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "all":
		printer.Info("Stopping Tilt...")
		return tiltmgr.Down()
	case "select":
		resources, err := tiltmgr.GetStatus()
		if err != nil {
			return err
		}
		var running []string
		for _, r := range resources {
			if r.Status == "ok" || r.Status == "pending" || r.Status == "error" {
				running = append(running, r.Name)
			}
		}
		if len(running) == 0 {
			printer.Info("No active resources to stop.")
			return nil
		}

		var selected []string
		options := make([]huh.Option[string], 0, len(running))
		for _, name := range running {
			options = append(options, huh.NewOption(name, name))
		}
		if err := huh.NewMultiSelect[string]().
			Title("Select resources to disable").
			Options(options...).
			Value(&selected).
			Run(); err != nil {
			return nil
		}
		if len(selected) == 0 {
			return nil
		}
		printer.Info(fmt.Sprintf("Disabling %d resources...", len(selected)))
		return tiltmgr.Disable(selected)
	}
	return nil
}

func runTiltEnableInteractive() error {
	printer := tui.NewStatusPrinter()
	if !tiltmgr.IsRunning() {
		printer.Info("Tilt is not running.")
		return nil
	}

	disabled, err := tiltmgr.GetDisabledResources()
	if err != nil {
		return err
	}
	if len(disabled) == 0 {
		printer.Info("No disabled resources.")
		return nil
	}

	var choice string
	if err := huh.NewSelect[string]().
		Title(fmt.Sprintf("Enable resources (%d disabled)", len(disabled))).
		Options(
			huh.NewOption("Enable all", "all"),
			huh.NewOption("Select resources", "select"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "all":
		printer.Info(fmt.Sprintf("Enabling %d resources...", len(disabled)))
		return tiltmgr.Enable(disabled)
	case "select":
		var selected []string
		options := make([]huh.Option[string], 0, len(disabled))
		for _, name := range disabled {
			options = append(options, huh.NewOption(name, name))
		}
		if err := huh.NewMultiSelect[string]().
			Title("Select resources to enable").
			Options(options...).
			Value(&selected).
			Run(); err != nil {
			return nil
		}
		if len(selected) > 0 {
			printer.Info(fmt.Sprintf("Enabling %d resources...", len(selected)))
			return tiltmgr.Enable(selected)
		}
	}
	return nil
}

func runTiltRetryInteractive() error {
	printer := tui.NewStatusPrinter()
	if !tiltmgr.IsRunning() {
		printer.Info("Tilt is not running.")
		return nil
	}

	errored, err := tiltmgr.GetErroredResources()
	if err != nil {
		return err
	}
	if len(errored) == 0 {
		printer.Info("No errored resources.")
		return nil
	}

	var choice string
	if err := huh.NewSelect[string]().
		Title(fmt.Sprintf("Retry errored resources (%d)", len(errored))).
		Options(
			huh.NewOption("Retry all", "all"),
			huh.NewOption("Select resources", "select"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "all":
		printer.Info(fmt.Sprintf("Retrying %d resources...", len(errored)))
		return tiltmgr.Retry(errored)
	case "select":
		var selected []string
		options := make([]huh.Option[string], 0, len(errored))
		for _, name := range errored {
			options = append(options, huh.NewOption(name, name))
		}
		if err := huh.NewMultiSelect[string]().
			Title("Select resources to retry").
			Options(options...).
			Value(&selected).
			Run(); err != nil {
			return nil
		}
		if len(selected) > 0 {
			printer.Info(fmt.Sprintf("Retrying %d resources...", len(selected)))
			return tiltmgr.Retry(selected)
		}
	}
	return nil
}

func runTiltProfileInteractive() error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("Profiles").
		Options(
			huh.NewOption("List profiles", "list"),
			huh.NewOption("Show profile details", "show"),
			huh.NewOption("Create profile", "create"),
			huh.NewOption("Edit profile", "edit"),
			huh.NewOption("Delete profile", "delete"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	projectRoot, _, err := findProject()
	if err != nil {
		return err
	}
	pm := tiltmgr.NewProfileManager(filepath.Join(projectRoot, ".tilt-profiles"))

	switch choice {
	case "list":
		return printProfileList(pm)
	case "show":
		return showProfileInteractive(pm)
	case "create":
		return createProfileInteractive(projectRoot, pm)
	case "edit":
		return editProfileInteractive(projectRoot, pm)
	case "delete":
		return deleteProfileInteractive(pm)
	}
	return nil
}

func showProfileInteractive(pm *tiltmgr.ProfileManager) error {
	profiles, err := pm.List()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		tui.NewStatusPrinter().Info("No profiles found.")
		return nil
	}

	options := make([]huh.Option[string], 0, len(profiles))
	for _, p := range profiles {
		options = append(options, huh.NewOption(p.Name, p.Name))
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Select profile to view").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return nil
	}

	profile, err := pm.Load(selected)
	if err != nil {
		return err
	}

	printProfileBox(profile)
	return nil
}

func printProfileBox(profile *tiltmgr.Profile) {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorPrimary).
		Padding(1, 2).
		Width(50)

	var content strings.Builder
	content.WriteString(tui.BoldStyle.Render(profile.Name) + "\n")
	if profile.Description != "" {
		content.WriteString(tui.MutedStyle.Render(profile.Description) + "\n")
	}
	content.WriteString("\n")

	if len(profile.Infra) > 0 {
		content.WriteString(tui.BoldStyle.Render("Infrastructure") + "\n")
		for _, r := range profile.Infra {
			content.WriteString("  " + r + "\n")
		}
		content.WriteString("\n")
	}
	if len(profile.Services) > 0 {
		content.WriteString(tui.BoldStyle.Render("Services") + "\n")
		for _, r := range profile.Services {
			content.WriteString("  " + r + "\n")
		}
	}

	total := len(profile.Infra) + len(profile.Services)
	content.WriteString("\n" + tui.MutedStyle.Render(fmt.Sprintf("Total: %d resources", total)))

	fmt.Println()
	fmt.Println(boxStyle.Render(content.String()))
}

func createProfileInteractive(projectRoot string, pm *tiltmgr.ProfileManager) error {
	var name string
	if err := huh.NewInput().
		Title("Profile name").
		Value(&name).
		Run(); err != nil {
		return nil
	}
	if name == "" {
		return nil
	}
	if pm.Exists(name) {
		return fmt.Errorf("profile %q already exists", name)
	}

	tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
	discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
	if err != nil {
		return err
	}

	profile, err := interactiveProfileSelect(name, "", discovered, nil)
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
}

func editProfileInteractive(projectRoot string, pm *tiltmgr.ProfileManager) error {
	profiles, err := pm.List()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		tui.NewStatusPrinter().Info("No profiles to edit.")
		return nil
	}

	options := make([]huh.Option[string], 0, len(profiles))
	for _, p := range profiles {
		options = append(options, huh.NewOption(p.Name, p.Name))
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Select profile to edit").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return nil
	}

	existing, err := pm.Load(selected)
	if err != nil {
		return err
	}

	tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
	discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
	if err != nil {
		return err
	}

	profile, err := interactiveProfileSelect(selected, existing.Description, discovered, existing)
	if err != nil {
		return err
	}

	if err := pm.Save(profile); err != nil {
		return err
	}

	printer := tui.NewStatusPrinter()
	printer.Success(fmt.Sprintf("Profile %q updated", selected))
	printProfileBox(profile)
	return nil
}

func deleteProfileInteractive(pm *tiltmgr.ProfileManager) error {
	profiles, err := pm.List()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		tui.NewStatusPrinter().Info("No profiles to delete.")
		return nil
	}

	options := make([]huh.Option[string], 0, len(profiles))
	for _, p := range profiles {
		options = append(options, huh.NewOption(p.Name, p.Name))
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Select profile to delete").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return nil
	}

	var confirm bool
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Delete profile %q?", selected)).
		Value(&confirm).
		Run(); err != nil {
		return nil
	}
	if !confirm {
		return nil
	}

	if err := pm.Delete(selected); err != nil {
		return err
	}
	tui.NewStatusPrinter().Success(fmt.Sprintf("Profile %q deleted", selected))
	return nil
}

func runTiltServiceInteractive() error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("Services").
		Options(
			huh.NewOption("List all services", "list"),
			huh.NewOption("Show by group", "groups"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	projectRoot, _, err := findProject()
	if err != nil {
		return err
	}

	tiltfilePath := filepath.Join(projectRoot, "Tiltfile")
	discovered, err := tiltmgr.ParseTiltfile(tiltfilePath)
	if err != nil {
		return err
	}

	switch choice {
	case "list":
		return printServiceList(discovered)
	case "groups":
		return printServiceGroups(discovered)
	}
	return nil
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

func printTiltStatus() error {
	printer := tui.NewStatusPrinter()

	if !tiltmgr.IsRunning() {
		printer.Info("Tilt is not running.")
		return nil
	}

	resources, err := tiltmgr.GetStatus()
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		printer.Info("No resources found.")
		return nil
	}

	// Summary counts
	summary, _ := tiltmgr.GetStatusSummary()
	if summary != nil {
		fmt.Println()
		parts := []string{
			fmt.Sprintf("Total: %d", summary.Total),
		}
		if summary.Running > 0 {
			parts = append(parts, tui.SuccessStyle.Render(fmt.Sprintf("● %d running", summary.Running)))
		}
		if summary.Pending > 0 {
			parts = append(parts, tui.WarningStyle.Render(fmt.Sprintf("◐ %d pending", summary.Pending)))
		}
		if summary.Error > 0 {
			parts = append(parts, tui.ErrorStyle.Render(fmt.Sprintf("✗ %d error", summary.Error)))
		}
		if summary.Disabled > 0 {
			parts = append(parts, tui.MutedStyle.Render(fmt.Sprintf("◌ %d disabled", summary.Disabled)))
		}
		fmt.Printf("  %s\n", strings.Join(parts, "  "))
	}

	// Resource table
	fmt.Printf("\n  %-30s %-12s %-15s %s\n", "NAME", "STATUS", "TYPE", "UPDATE")
	fmt.Printf("  %s\n", strings.Repeat("─", 70))

	for _, r := range resources {
		icon, statusStyle := tiltStatusStyle(r.Status)
		fmt.Printf("  %s %-28s %s  %-15s %s\n",
			statusStyle.Render(icon),
			r.Name,
			statusStyle.Render(fmt.Sprintf("%-10s", r.Status)),
			r.Type,
			r.UpdateStatus,
		)
	}
	fmt.Println()
	return nil
}

func tiltStatusStyle(status string) (string, lipgloss.Style) {
	switch status {
	case "ok":
		return "●", tui.SuccessStyle
	case "pending":
		return "◐", tui.WarningStyle
	case "error":
		return "✗", tui.ErrorStyle
	case "disabled":
		return "◌", tui.MutedStyle
	default:
		return "○", tui.MutedStyle
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

func printProfileList(pm *tiltmgr.ProfileManager) error {
	profiles, err := pm.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		tui.NewStatusPrinter().Info("No profiles found. Create one with: iron tilt profile create <name>")
		return nil
	}

	fmt.Printf("\n  %-20s %-10s %-10s %s\n", "NAME", "SERVICES", "INFRA", "DESCRIPTION")
	fmt.Printf("  %s\n", strings.Repeat("─", 60))

	for _, p := range profiles {
		fmt.Printf("  %-20s %-10d %-10d %s\n",
			p.Name, len(p.Services), len(p.Infra), p.Description)
	}
	fmt.Printf("\n  %s\n\n", tui.MutedStyle.Render(fmt.Sprintf("Total: %d profiles", len(profiles))))
	return nil
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

func printServiceList(discovered *tiltmgr.DiscoveredResources) error {
	fmt.Printf("\n  %-30s %-15s %s\n", "NAME", "GROUP", "TYPE")
	fmt.Printf("  %s\n", strings.Repeat("─", 55))

	for _, name := range discovered.Infra {
		fmt.Printf("  %-30s %-15s %s\n", name, "infra", tui.MutedStyle.Render("infra"))
	}
	for _, svc := range discovered.Services {
		fmt.Printf("  %-30s %-15s %s\n", svc.Name, svc.Group, "service")
	}

	fmt.Printf("\n  %s\n\n",
		tui.MutedStyle.Render(fmt.Sprintf("%d services, %d infra resources",
			len(discovered.Services), len(discovered.Infra))))
	return nil
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

func printServiceGroups(discovered *tiltmgr.DiscoveredResources) error {
	groups := make(map[string][]string)
	if len(discovered.Infra) > 0 {
		groups["infra"] = discovered.Infra
	}
	for _, svc := range discovered.Services {
		groups[svc.Group] = append(groups[svc.Group], svc.Name)
	}

	fmt.Println()
	for group, members := range groups {
		fmt.Printf("  %s %s\n",
			tui.BoldStyle.Render(group),
			tui.MutedStyle.Render(fmt.Sprintf("(%d)", len(members))))
		for _, m := range members {
			fmt.Printf("    • %s\n", m)
		}
		fmt.Println()
	}
	return nil
}

// --- helpers ---

// findProject locates the project root and loads the config.
func findProject() (string, *config.ProjectConfig, error) {
	cfgPath, err := config.FindConfigFile(".")
	if err != nil {
		return "", nil, fmt.Errorf("not in an ironplate project: %w", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return "", nil, err
	}
	return filepath.Dir(cfgPath), cfg, nil
}

// interactiveProfileSelect runs the interactive profile creation/edit flow.
func interactiveProfileSelect(name, description string, discovered *tiltmgr.DiscoveredResources, existing *tiltmgr.Profile) (*tiltmgr.Profile, error) {
	var selectedInfra []string
	if len(discovered.Infra) > 0 {
		infraOptions := make([]huh.Option[string], 0, len(discovered.Infra))
		for _, name := range discovered.Infra {
			infraOptions = append(infraOptions, huh.NewOption(name, name))
		}

		if existing == nil {
			selectedInfra = append([]string{}, discovered.Infra...)
		} else {
			selectedInfra = append([]string{}, existing.Infra...)
		}

		if err := huh.NewMultiSelect[string]().
			Title("Select infrastructure resources").
			Options(infraOptions...).
			Value(&selectedInfra).
			Run(); err != nil {
			return nil, err
		}
	}

	var selectedServices []string
	if len(discovered.Services) > 0 {
		svcOptions := make([]huh.Option[string], 0, len(discovered.Services))
		for _, svc := range discovered.Services {
			label := svc.Name
			if svc.Group != "default" {
				label = fmt.Sprintf("%s (%s)", svc.Name, svc.Group)
			}
			svcOptions = append(svcOptions, huh.NewOption(label, svc.Name))
		}

		if existing != nil {
			selectedServices = append([]string{}, existing.Services...)
		}

		if err := huh.NewMultiSelect[string]().
			Title("Select application services").
			Options(svcOptions...).
			Value(&selectedServices).
			Run(); err != nil {
			return nil, err
		}
	}

	if description == "" {
		if err := huh.NewInput().
			Title("Profile description (optional)").
			Value(&description).
			Run(); err != nil {
			return nil, err
		}
	}

	return &tiltmgr.Profile{
		Name:        name,
		Description: description,
		Services:    selectedServices,
		Infra:       selectedInfra,
	}, nil
}

func chdir(dir string) error {
	return os.Chdir(dir)
}
