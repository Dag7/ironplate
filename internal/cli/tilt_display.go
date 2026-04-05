package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/tiltmgr"
	"github.com/dag7/ironplate/internal/tui"
)

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

func printProfileList(pm *tiltmgr.ProfileManager) error {
	profiles, err := pm.List()
	if err != nil {
		return err
	}

	if len(profiles) == 0 {
		tui.NewStatusPrinter().Info("No profiles found in tilt/profiles.yaml")
		return nil
	}

	active, _ := pm.ActiveProfile()

	fmt.Printf("\n  %-15s %-15s %-15s %s\n", "NAME", "SERVICES", "INFRA", "DESCRIPTION")
	fmt.Printf("  %s\n", strings.Repeat("─", 70))

	for _, p := range profiles {
		marker := "  "
		if p.Name == active {
			marker = "→ "
		}
		svcDisplay := tiltmgr.FormatServicesDisplay(p.ServicesRaw)
		infraDisplay := tiltmgr.FormatInfraDisplay(p.InfraRaw)
		fmt.Printf("%s%-15s %-15s %-15s %s\n",
			marker, p.Name, svcDisplay, infraDisplay, tui.MutedStyle.Render(p.Description))
	}

	fmt.Printf("\n  %s\n\n", tui.MutedStyle.Render(
		fmt.Sprintf("Active: %s  |  %d profiles  |  Set with: iron tilt profile set <name>", active, len(profiles))))
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

	content.WriteString(tui.BoldStyle.Render("Services: ") +
		tiltmgr.FormatServicesDisplay(profile.ServicesRaw) + "\n")
	content.WriteString(tui.BoldStyle.Render("Infra:    ") +
		tiltmgr.FormatInfraDisplay(profile.InfraRaw) + "\n")

	fmt.Println()
	fmt.Println(boxStyle.Render(content.String()))
}

func printServiceList(discovered *tiltmgr.DiscoveredResources) error {
	fmt.Printf("\n  %-25s %-12s %-12s %-8s %s\n", "NAME", "GROUP", "TYPE", "PORT", "LABELS")
	fmt.Printf("  %s\n", strings.Repeat("─", 70))

	for _, svc := range discovered.Services {
		labels := strings.Join(svc.Labels, ", ")
		fmt.Printf("  %-25s %-12s %-12s %-8d %s\n",
			svc.Name, svc.Group, svc.Type, svc.Port, tui.MutedStyle.Render(labels))
	}

	fmt.Println()
	for _, infra := range discovered.Infra {
		status := tui.MutedStyle.Render("disabled")
		if infra.Enabled {
			status = tui.SuccessStyle.Render("enabled")
		}
		deps := ""
		if len(infra.Deps) > 0 {
			deps = tui.MutedStyle.Render("deps: " + strings.Join(infra.Deps, ", "))
		}
		tags := ""
		if infra.Required {
			tags = tui.WarningStyle.Render("[required]")
		}
		if infra.Local {
			tags += " " + tui.MutedStyle.Render("[local]")
		}
		fmt.Printf("  %-25s %-12s %-12s %s %s\n", infra.Name, "infra", status, tags, deps)
	}

	fmt.Printf("\n  %s\n\n",
		tui.MutedStyle.Render(fmt.Sprintf("%d services, %d infra components",
			len(discovered.Services), len(discovered.Infra))))
	return nil
}

func printServiceGroups(discovered *tiltmgr.DiscoveredResources) error {
	groups := make(map[string][]string)
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

	if len(discovered.Infra) > 0 {
		fmt.Printf("  %s %s\n",
			tui.BoldStyle.Render("infrastructure"),
			tui.MutedStyle.Render(fmt.Sprintf("(%d)", len(discovered.Infra))))
		for _, infra := range discovered.Infra {
			if infra.Enabled {
				fmt.Printf("    • %s\n", infra.Name)
			}
		}
		fmt.Println()
	}

	return nil
}
