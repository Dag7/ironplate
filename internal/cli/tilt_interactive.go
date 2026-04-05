package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/tiltmgr"
	"github.com/dag7/ironplate/internal/tui"
)

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

	pm := tiltmgr.NewProfileManager(projectRoot)
	profiles, err := pm.List()
	if err != nil {
		return err
	}

	active, _ := pm.ActiveProfile()

	// Build options: profiles first
	options := make([]huh.Option[string], 0, len(profiles))
	for _, p := range profiles {
		label := p.Name
		if p.Description != "" {
			label = fmt.Sprintf("%s — %s", p.Name, p.Description)
		}
		if p.Name == active {
			label = label + " (active)"
		}
		options = append(options, huh.NewOption(label, p.Name))
	}

	var choice string
	if err := huh.NewSelect[string]().
		Title("Start Tilt with profile").
		Options(options...).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	printer := tui.NewStatusPrinter()

	profile, err := pm.Load(choice)
	if err != nil {
		return err
	}

	printer.Info(fmt.Sprintf("Starting Tilt with profile %q...", choice))

	if err := os.Chdir(projectRoot); err != nil {
		return err
	}

	_ = profile // validated above
	return tiltmgr.Up(choice, false)
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
		projectRoot, _, err := findProject()
		if err != nil {
			return err
		}
		if err := os.Chdir(projectRoot); err != nil {
			return err
		}
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
			huh.NewOption("Set active profile", "set"),
			huh.NewOption("Create profile", "create"),
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
	pm := tiltmgr.NewProfileManager(projectRoot)

	switch choice {
	case "list":
		return printProfileList(pm)
	case "show":
		return showProfileInteractive(pm)
	case "set":
		return setProfileInteractive(pm)
	case "create":
		return createProfileInteractive(projectRoot, pm)
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
		label := p.Name
		if p.Description != "" {
			label = fmt.Sprintf("%s — %s", p.Name, p.Description)
		}
		options = append(options, huh.NewOption(label, p.Name))
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

func setProfileInteractive(pm *tiltmgr.ProfileManager) error {
	profiles, err := pm.List()
	if err != nil {
		return err
	}
	active, _ := pm.ActiveProfile()

	options := make([]huh.Option[string], 0, len(profiles))
	for _, p := range profiles {
		label := p.Name
		if p.Name == active {
			label = p.Name + " (current)"
		}
		if p.Description != "" {
			label = label + " — " + p.Description
		}
		options = append(options, huh.NewOption(label, p.Name))
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Set active profile").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return nil
	}

	if err := pm.SetActive(selected); err != nil {
		return err
	}
	tui.NewStatusPrinter().Success(fmt.Sprintf("Active profile set to %q", selected))
	return nil
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

	discovered, err := tiltmgr.Discover(projectRoot)
	if err != nil {
		return err
	}

	profile, svcList, infraList, err := interactiveProfileSelect(name, "", discovered, nil)
	if err != nil {
		return err
	}

	if err := pm.Save(name, profile.Description, svcList, infraList); err != nil {
		return err
	}

	printer := tui.NewStatusPrinter()
	printer.Success(fmt.Sprintf("Profile %q created", name))
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

	discovered, err := tiltmgr.Discover(projectRoot)
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

// interactiveProfileSelect runs the interactive profile creation flow.
// Returns the profile for display, plus the raw services/infra lists for saving.
func interactiveProfileSelect(name, description string, discovered *tiltmgr.DiscoveredResources, _ *tiltmgr.Profile) (*tiltmgr.Profile, []string, []string, error) {
	// Select infrastructure
	var selectedInfra []string
	if len(discovered.Infra) > 0 {
		infraOptions := make([]huh.Option[string], 0, len(discovered.Infra))
		for _, infra := range discovered.Infra {
			label := infra.Name
			if infra.Required {
				label += " (required)"
			}
			infraOptions = append(infraOptions, huh.NewOption(label, infra.Name))
		}

		// Default: all enabled infra selected
		for _, infra := range discovered.Infra {
			if infra.Enabled {
				selectedInfra = append(selectedInfra, infra.Name)
			}
		}

		if err := huh.NewMultiSelect[string]().
			Title("Select infrastructure").
			Options(infraOptions...).
			Value(&selectedInfra).
			Run(); err != nil {
			return nil, nil, nil, err
		}
	}

	// Select services
	var selectedServices []string
	if len(discovered.Services) > 0 {
		svcOptions := make([]huh.Option[string], 0, len(discovered.Services))
		for _, svc := range discovered.Services {
			label := fmt.Sprintf("%s (%s, %s)", svc.Name, svc.Group, svc.Type)
			svcOptions = append(svcOptions, huh.NewOption(label, svc.Name))
		}

		if err := huh.NewMultiSelect[string]().
			Title("Select services").
			Options(svcOptions...).
			Value(&selectedServices).
			Run(); err != nil {
			return nil, nil, nil, err
		}
	}

	// Description
	if description == "" {
		if err := huh.NewInput().
			Title("Profile description (optional)").
			Value(&description).
			Run(); err != nil {
			return nil, nil, nil, err
		}
	}

	// Build the profile for display
	svcInterface := make([]interface{}, len(selectedServices))
	for i, s := range selectedServices {
		svcInterface[i] = s
	}
	infraInterface := make([]interface{}, len(selectedInfra))
	for i, s := range selectedInfra {
		infraInterface[i] = s
	}

	profile := &tiltmgr.Profile{
		Name:        name,
		Description: description,
		ServicesRaw: svcInterface,
		InfraRaw:    infraInterface,
	}

	return profile, selectedServices, selectedInfra, nil
}
