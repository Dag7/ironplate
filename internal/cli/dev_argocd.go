package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/devtools"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newDevArgoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "argocd",
		Aliases: []string{"argo"},
		Short:   "ArgoCD GitOps management",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List ArgoCD applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listArgoApps()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "sync [app]",
		Short: "Sync an ArgoCD application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()
			appName := ""
			if len(args) > 0 {
				appName = args[0]
			} else {
				var err error
				appName, err = selectArgoApp()
				if err != nil || appName == "" {
					return err
				}
			}

			// Production safeguard
			if err := checkProductionArgoGuard(); err != nil {
				return err
			}

			printer.Info("Syncing " + appName + "...")
			if err := devtools.SyncArgoApp(appName); err != nil {
				return err
			}
			printer.Success("Sync triggered for " + appName)
			return nil
		},
	})

	cmd.AddCommand(newDevArgoRefreshCmd())
	cmd.AddCommand(newDevArgoStatusCmd())
	cmd.AddCommand(newDevArgoSyncMultipleCmd())

	return cmd
}

func newDevArgoRefreshCmd() *cobra.Command {
	var hard bool
	cmd := &cobra.Command{
		Use:   "refresh [app]",
		Short: "Refresh an ArgoCD application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()
			appName := ""
			if len(args) > 0 {
				appName = args[0]
			} else {
				var err error
				appName, err = selectArgoApp()
				if err != nil || appName == "" {
					return err
				}
			}
			printer.Info("Refreshing " + appName + "...")
			if err := devtools.RefreshArgoApp(appName, hard); err != nil {
				return err
			}
			printer.Success("Refreshed " + appName)
			return nil
		},
	}
	cmd.Flags().BoolVar(&hard, "hard", false, "Hard refresh (clear cache)")
	return cmd
}

func newDevArgoStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [app]",
		Short: "Show detailed status of an ArgoCD application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appName := ""
			if len(args) > 0 {
				appName = args[0]
			} else {
				var err error
				appName, err = selectArgoApp()
				if err != nil || appName == "" {
					return err
				}
			}
			return showArgoAppStatus(appName)
		},
	}
}

func newDevArgoSyncMultipleCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:     "sync-multiple",
		Aliases: []string{"sync-all"},
		Short:   "Sync multiple ArgoCD applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := tui.NewStatusPrinter()

			// Production safeguard
			if err := checkProductionArgoGuard(); err != nil {
				return err
			}

			if all {
				apps, err := devtools.ListArgoApps()
				if err != nil {
					return err
				}
				if len(apps) == 0 {
					printer.Info("No ArgoCD applications found.")
					return nil
				}

				var toSync []string
				for _, app := range apps {
					toSync = append(toSync, app.Name)
				}

				// Confirmation
				var confirm bool
				if err := huh.NewConfirm().
					Title(fmt.Sprintf("Sync %d applications?", len(toSync))).
					Value(&confirm).
					Run(); err != nil {
					return nil
				}
				if !confirm {
					return nil
				}

				results := devtools.SyncMultipleArgoApps(toSync)
				for name, err := range results {
					if err != nil {
						printer.Error(fmt.Sprintf("%s: %s", name, err.Error()))
					} else {
						printer.Success(fmt.Sprintf("Sync triggered: %s", name))
					}
				}
				return nil
			}

			return runSyncMultipleInteractive()
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Sync all applications without prompting")
	return cmd
}

func checkProductionArgoGuard() error {
	ctx, err := devtools.GetCurrentContext()
	if err != nil {
		return nil // Can't determine context, proceed
	}
	lower := strings.ToLower(ctx)
	if strings.Contains(lower, "prod") || strings.Contains(lower, "prd") {
		printer := tui.NewStatusPrinter()
		fmt.Println()
		printer.Warning("You are connected to a PRODUCTION context!")

		var confirm bool
		if err := huh.NewConfirm().
			Title("Continue with production ArgoCD operation?").
			Value(&confirm).
			Run(); err != nil {
			return fmt.Errorf("cancelled")
		}
		if !confirm {
			return fmt.Errorf("cancelled")
		}
	}
	return nil
}

func argoSyncStyle(status string) lipgloss.Style {
	if status == "OutOfSync" {
		return tui.WarningStyle
	}
	return tui.SuccessStyle
}

func argoHealthStyle(status string) lipgloss.Style {
	switch status {
	case "Progressing":
		return tui.WarningStyle
	case "Degraded", "Missing":
		return tui.ErrorStyle
	default:
		return tui.SuccessStyle
	}
}

func showArgoAppStatus(appName string) error {
	detail, err := devtools.GetArgoAppStatus(appName)
	if err != nil {
		return err
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorPrimary).
		Padding(1, 2).
		Width(70)

	var content strings.Builder
	content.WriteString(tui.BoldStyle.Render(detail.Name) + "\n\n")
	fmt.Fprintf(&content, "  Project:    %s\n", detail.Project)
	fmt.Fprintf(&content, "  Repo:       %s\n", detail.RepoURL)
	fmt.Fprintf(&content, "  Path:       %s\n", detail.Path)
	fmt.Fprintf(&content, "  Revision:   %s\n", detail.Revision)

	syncStyle := argoSyncStyle(detail.SyncStatus)
	healthStyle := argoHealthStyle(detail.HealthStatus)

	fmt.Fprintf(&content, "  Sync:       %s %s\n",
		devtools.SyncIcon(detail.SyncStatus),
		syncStyle.Render(detail.SyncStatus))
	fmt.Fprintf(&content, "  Health:     %s %s\n",
		devtools.HealthIcon(detail.HealthStatus),
		healthStyle.Render(detail.HealthStatus))

	if detail.LastSyncTime != "" {
		fmt.Fprintf(&content, "  Last sync:  %s\n", detail.LastSyncTime)
	}

	fmt.Println()
	fmt.Println(boxStyle.Render(content.String()))

	// Show resources (up to 10)
	if len(detail.Resources) > 0 {
		fmt.Printf("\n  %s\n", tui.BoldStyle.Render("Resources"))
		fmt.Printf("  %-20s %-30s %-15s %s\n", "KIND", "NAME", "STATUS", "HEALTH")
		fmt.Printf("  %s\n", strings.Repeat("─", 70))

		limit := len(detail.Resources)
		if limit > 10 {
			limit = 10
		}
		for _, r := range detail.Resources[:limit] {
			health := r.Health
			if health == "" {
				health = "-"
			}
			fmt.Printf("  %-20s %-30s %-15s %s\n", r.Kind, r.Name, r.Status, health)
		}
		if len(detail.Resources) > 10 {
			fmt.Printf("  %s\n", tui.MutedStyle.Render(fmt.Sprintf("... and %d more resources", len(detail.Resources)-10)))
		}
		fmt.Println()
	}
	return nil
}

func runArgoInteractive() error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("ArgoCD").
		Options(
			huh.NewOption("List applications", "list"),
			huh.NewOption("Application status", "status"),
			huh.NewOption("Sync application", "sync"),
			huh.NewOption("Sync multiple", "sync-multiple"),
			huh.NewOption("Refresh application", "refresh"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	printer := tui.NewStatusPrinter()
	switch choice {
	case "list":
		return listArgoApps()
	case "status":
		app, err := selectArgoApp()
		if err != nil || app == "" {
			return err
		}
		return showArgoAppStatus(app)
	case "sync":
		app, err := selectArgoApp()
		if err != nil || app == "" {
			return err
		}
		if err := checkProductionArgoGuard(); err != nil {
			return err
		}
		printer.Info("Syncing " + app + "...")
		if err := devtools.SyncArgoApp(app); err != nil {
			return err
		}
		printer.Success("Sync triggered for " + app)
	case "sync-multiple":
		return runSyncMultipleInteractive()
	case "refresh":
		app, err := selectArgoApp()
		if err != nil || app == "" {
			return err
		}
		printer.Info("Refreshing " + app + "...")
		if err := devtools.RefreshArgoApp(app, false); err != nil {
			return err
		}
		printer.Success("Refreshed " + app)
	}
	return nil
}

func runSyncMultipleInteractive() error {
	printer := tui.NewStatusPrinter()

	if err := checkProductionArgoGuard(); err != nil {
		return err
	}

	apps, err := devtools.ListArgoApps()
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		printer.Info("No ArgoCD applications found.")
		return nil
	}

	options := make([]huh.Option[string], 0, len(apps))
	var preSelected []string
	for _, app := range apps {
		label := fmt.Sprintf("%s %s (%s / %s)",
			devtools.SyncIcon(app.SyncStatus), app.Name,
			app.SyncStatus, app.HealthStatus)
		options = append(options, huh.NewOption(label, app.Name))
		if app.SyncStatus == "OutOfSync" {
			preSelected = append(preSelected, app.Name)
		}
	}

	toSync := preSelected
	if err := huh.NewMultiSelect[string]().
		Title("Select applications to sync").
		Options(options...).
		Value(&toSync).
		Run(); err != nil {
		return nil
	}

	if len(toSync) == 0 {
		printer.Info("No applications selected.")
		return nil
	}

	var confirm bool
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Sync %d applications?", len(toSync))).
		Value(&confirm).
		Run(); err != nil {
		return nil
	}
	if !confirm {
		return nil
	}

	results := devtools.SyncMultipleArgoApps(toSync)
	for name, syncErr := range results {
		if syncErr != nil {
			printer.Error(fmt.Sprintf("%s: %s", name, syncErr.Error()))
		} else {
			printer.Success(fmt.Sprintf("Sync triggered: %s", name))
		}
	}
	return nil
}

func listArgoApps() error {
	apps, err := devtools.ListArgoApps()
	if err != nil {
		return err
	}

	if len(apps) == 0 {
		tui.NewStatusPrinter().Info("No ArgoCD applications found.")
		return nil
	}

	grouped := devtools.GroupByProject(apps)
	fmt.Println()
	for project, projectApps := range grouped {
		fmt.Printf("  %s\n", tui.BoldStyle.Render(project))
		for _, app := range projectApps {
			syncStyle := argoSyncStyle(app.SyncStatus)
			healthStyle := argoHealthStyle(app.HealthStatus)

			fmt.Printf("    %s %s  %-30s  %s  %s\n",
				healthStyle.Render(devtools.HealthIcon(app.HealthStatus)),
				syncStyle.Render(devtools.SyncIcon(app.SyncStatus)),
				app.Name,
				syncStyle.Render(app.SyncStatus),
				healthStyle.Render(app.HealthStatus),
			)
		}
	}
	fmt.Println()
	return nil
}

func selectArgoApp() (string, error) {
	apps, err := devtools.ListArgoApps()
	if err != nil {
		return "", err
	}
	if len(apps) == 0 {
		return "", fmt.Errorf("no ArgoCD applications found")
	}

	options := make([]huh.Option[string], 0, len(apps))
	for _, app := range apps {
		label := fmt.Sprintf("%s %s (%s / %s)",
			devtools.SyncIcon(app.SyncStatus), app.Name, app.SyncStatus, app.HealthStatus)
		options = append(options, huh.NewOption(label, app.Name))
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Select application").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return "", nil
	}
	return selected, nil
}
