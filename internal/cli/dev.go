package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/devtools"
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

// --- iron dev context ---

func newDevContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "context",
		Aliases: []string{"ctx"},
		Short:   "Kubernetes context management",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "local",
		Short: "Switch to local k3d context",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			return switchToContext(devtools.GetLocalContextName(cfg.Metadata.Name), "local")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "staging",
		Aliases: []string{"stg"},
		Short:   "Switch to staging context",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			return switchToStagingContext(cfg)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "production",
		Aliases: []string{"prod"},
		Short:   "Switch to production context (requires confirmation)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			return switchToProductionContext(cfg)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "current",
		Short: "Show current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := devtools.GetCurrentContext()
			if err != nil {
				return err
			}
			envBadge := contextEnvBadge(ctx)
			fmt.Printf("  Current context: %s %s\n", tui.BoldStyle.Render(ctx), envBadge)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			contexts, err := devtools.ListContexts()
			if err != nil {
				return err
			}
			fmt.Printf("\n  %-5s %-40s %-20s %s\n", "", "CONTEXT", "CLUSTER", "ENV")
			fmt.Printf("  %s\n", strings.Repeat("─", 70))
			for _, ctx := range contexts {
				marker := "  "
				if ctx.Current {
					marker = tui.SuccessStyle.Render("* ")
				}
				envBadge := contextEnvBadge(ctx.Name)
				fmt.Printf("  %s%-40s %-20s %s\n", marker, ctx.Name, ctx.Cluster, envBadge)
			}
			fmt.Println()
			return nil
		},
	})

	return cmd
}

func contextEnvBadge(contextName string) string {
	lower := strings.ToLower(contextName)
	switch {
	case strings.Contains(lower, "k3d") || strings.Contains(lower, "local"):
		return tui.SuccessStyle.Render("[local]")
	case strings.Contains(lower, "staging") || strings.Contains(lower, "stg"):
		return tui.WarningStyle.Render("[staging]")
	case strings.Contains(lower, "prod") || strings.Contains(lower, "prd"):
		return tui.ErrorStyle.Render("[production]")
	default:
		return ""
	}
}

func switchToStagingContext(cfg *config.ProjectConfig) error {
	// Try to find a staging context by convention
	contexts, err := devtools.ListContexts()
	if err != nil {
		return err
	}

	var stagingCtx string
	for _, ctx := range contexts {
		lower := strings.ToLower(ctx.Name)
		if strings.Contains(lower, "staging") || strings.Contains(lower, "stg") {
			stagingCtx = ctx.Name
			break
		}
	}

	if stagingCtx == "" {
		// Try GKE naming convention
		for _, env := range cfg.Spec.Cloud.Environments {
			if env.Name == "staging" || env.ShortName == "stg" {
				for _, ctx := range contexts {
					if strings.Contains(ctx.Name, env.Name) || strings.Contains(ctx.Name, env.ShortName) {
						stagingCtx = ctx.Name
						break
					}
				}
			}
		}
	}

	if stagingCtx == "" {
		return fmt.Errorf("no staging context found. Use 'iron dev context list' to see available contexts")
	}

	return switchToContext(stagingCtx, "staging")
}

func switchToProductionContext(cfg *config.ProjectConfig) error {
	printer := tui.NewStatusPrinter()

	// Production safeguard
	fmt.Println()
	printer.Warning("You are about to switch to PRODUCTION context")

	var confirm bool
	if err := huh.NewConfirm().
		Title("Are you sure you want to switch to production?").
		Value(&confirm).
		Run(); err != nil {
		return nil
	}
	if !confirm {
		printer.Info("Cancelled.")
		return nil
	}

	contexts, err := devtools.ListContexts()
	if err != nil {
		return err
	}

	var prodCtx string
	for _, ctx := range contexts {
		lower := strings.ToLower(ctx.Name)
		if strings.Contains(lower, "production") || strings.Contains(lower, "prod") || strings.Contains(lower, "prd") {
			prodCtx = ctx.Name
			break
		}
	}

	if prodCtx == "" {
		for _, env := range cfg.Spec.Cloud.Environments {
			if env.Name == "production" || env.ShortName == "prd" {
				for _, ctx := range contexts {
					if strings.Contains(ctx.Name, env.Name) || strings.Contains(ctx.Name, env.ShortName) {
						prodCtx = ctx.Name
						break
					}
				}
			}
		}
	}

	if prodCtx == "" {
		return fmt.Errorf("no production context found. Use 'iron dev context list' to see available contexts")
	}

	return switchToContext(prodCtx, "production")
}

func runContextInteractive(projectName string) error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("Context").
		Options(
			huh.NewOption("Switch to local (k3d)", "local"),
			huh.NewOption("Switch to staging", "staging"),
			huh.NewOption("Switch to production", "production"),
			huh.NewOption("Show current context", "current"),
			huh.NewOption("List all contexts", "list"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	_, cfg, _ := findProject()
	printer := tui.NewStatusPrinter()

	switch choice {
	case "local":
		return switchToContext(devtools.GetLocalContextName(projectName), "local")
	case "staging":
		if cfg != nil {
			return switchToStagingContext(cfg)
		}
		return fmt.Errorf("project config not found")
	case "production":
		if cfg != nil {
			return switchToProductionContext(cfg)
		}
		return fmt.Errorf("project config not found")
	case "current":
		ctx, err := devtools.GetCurrentContext()
		if err != nil {
			return err
		}
		envBadge := contextEnvBadge(ctx)
		printer.Info("Current: " + ctx + " " + envBadge)
	case "list":
		contexts, err := devtools.ListContexts()
		if err != nil {
			return err
		}
		for _, ctx := range contexts {
			marker := "  "
			if ctx.Current {
				marker = "* "
			}
			envBadge := contextEnvBadge(ctx.Name)
			fmt.Printf("  %s%s %s\n", marker, ctx.Name, envBadge)
		}
	}
	return nil
}

func switchToContext(contextName, envLabel string) error {
	printer := tui.NewStatusPrinter()
	if !devtools.ContextExists(contextName) {
		return fmt.Errorf("context %q not found. Is the cluster running?", contextName)
	}
	if err := devtools.SwitchContext(contextName); err != nil {
		return err
	}
	printer.Success(fmt.Sprintf("Switched to %s context: %s", envLabel, contextName))
	return nil
}

// --- iron dev pods ---

func newDevPodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pods",
		Aliases: []string{"pod"},
		Short:   "Pod management and debugging",
	}

	cmd.AddCommand(&cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pods grouped by service",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			return listPodsGrouped(cfg.Metadata.Name)
		},
	})

	cmd.AddCommand(newDevPodsLogsCmd())
	cmd.AddCommand(newDevPodsExecCmd())
	cmd.AddCommand(newDevPodsPortForwardCmd())
	cmd.AddCommand(newDevPodsDescribeCmd())

	return cmd
}

func newDevPodsLogsCmd() *cobra.Command {
	var (
		follow    bool
		tail      int
		container string
	)
	cmd := &cobra.Command{
		Use:   "logs [pod]",
		Short: "View pod logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			podName := ""
			if len(args) > 0 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}
			return devtools.PodLogs(podName, cfg.Metadata.Name, follow, tail, container)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&tail, "tail", "t", 100, "Number of lines from the end")
	cmd.Flags().StringVarP(&container, "container", "c", "", "Container name")
	return cmd
}

func newDevPodsExecCmd() *cobra.Command {
	var container string
	cmd := &cobra.Command{
		Use:     "exec [pod]",
		Aliases: []string{"sh"},
		Short:   "Shell into a pod",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			podName := ""
			if len(args) > 0 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}
			return devtools.PodExec(podName, cfg.Metadata.Name, container)
		},
	}
	cmd.Flags().StringVarP(&container, "container", "c", "", "Container name")
	return cmd
}

func newDevPodsPortForwardCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "port-forward [pod] [local:remote]",
		Aliases: []string{"pf"},
		Short:   "Forward ports to a pod",
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}

			podName := ""
			if len(args) >= 1 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}

			var local, remote int
			if len(args) >= 2 {
				if _, err := fmt.Sscanf(args[1], "%d:%d", &local, &remote); err != nil {
					return fmt.Errorf("invalid port format, use local:remote (e.g. 8080:80)")
				}
			} else {
				// Interactive port input
				var ports string
				if err := huh.NewInput().
					Title("Ports (local:remote, e.g. 8080:80)").
					Value(&ports).
					Run(); err != nil {
					return nil
				}
				if _, err := fmt.Sscanf(ports, "%d:%d", &local, &remote); err != nil {
					return fmt.Errorf("invalid port format, use local:remote (e.g. 8080:80)")
				}
			}

			return devtools.PodPortForward(podName, cfg.Metadata.Name, local, remote)
		},
	}
}

func newDevPodsDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "describe [pod]",
		Short: "Describe a pod",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			podName := ""
			if len(args) > 0 {
				podName = args[0]
			} else {
				podName, err = selectPod(cfg.Metadata.Name)
				if err != nil || podName == "" {
					return err
				}
			}
			return devtools.DescribePod(podName, cfg.Metadata.Name)
		},
	}
}

func runPodsInteractive(namespace string) error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("Pods").
		Options(
			huh.NewOption("List pods", "list"),
			huh.NewOption("View logs", "logs"),
			huh.NewOption("Shell into pod", "exec"),
			huh.NewOption("Port forward", "port-forward"),
			huh.NewOption("Describe pod", "describe"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "list":
		return listPodsGrouped(namespace)
	case "logs":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		// Tail lines selection
		var tailLines int
		if err := huh.NewSelect[int]().
			Title("Tail lines").
			Options(
				huh.NewOption("50 lines", 50),
				huh.NewOption("100 lines", 100),
				huh.NewOption("500 lines", 500),
				huh.NewOption("1000 lines", 1000),
			).
			Value(&tailLines).
			Run(); err != nil {
			tailLines = 100
		}
		return devtools.PodLogs(pod, namespace, true, tailLines, "")
	case "exec":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		return devtools.PodExec(pod, namespace, "")
	case "port-forward":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		var ports string
		if err := huh.NewInput().
			Title("Ports (local:remote, e.g. 8080:80)").
			Value(&ports).
			Run(); err != nil {
			return nil
		}
		var local, remote int
		if _, err := fmt.Sscanf(ports, "%d:%d", &local, &remote); err != nil {
			return fmt.Errorf("invalid port format, use local:remote")
		}
		return devtools.PodPortForward(pod, namespace, local, remote)
	case "describe":
		pod, err := selectPod(namespace)
		if err != nil || pod == "" {
			return err
		}
		return devtools.DescribePod(pod, namespace)
	}
	return nil
}

func listPodsGrouped(namespace string) error {
	pods, err := devtools.ListPods(namespace)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		tui.NewStatusPrinter().Info("No pods found in namespace " + namespace)
		return nil
	}

	groups := devtools.GroupPodsByService(pods)

	fmt.Println()
	for _, g := range groups {
		fmt.Printf("  %s\n", tui.BoldStyle.Render(g.Name))
		for _, p := range g.Pods {
			statusStyle := tui.MutedStyle
			switch strings.ToLower(p.Status) {
			case "running":
				statusStyle = tui.SuccessStyle
			case "pending":
				statusStyle = tui.WarningStyle
			case "failed", "error", "crashloopbackoff":
				statusStyle = tui.ErrorStyle
			}

			restartInfo := ""
			if p.Restarts > 0 {
				restartInfo = tui.WarningStyle.Render(fmt.Sprintf(" (%d restarts)", p.Restarts))
			}

			readyInfo := ""
			if p.Ready != "" && p.Ready != "<none>" {
				readyInfo = tui.MutedStyle.Render(fmt.Sprintf(" [ready:%s]", p.Ready))
			}

			fmt.Printf("    %s %-45s %s%s%s\n",
				statusStyle.Render(devtools.StatusIcon(p.Status)),
				p.Name,
				statusStyle.Render(p.Status),
				restartInfo,
				readyInfo,
			)
		}
	}
	fmt.Println()
	return nil
}

func selectPod(namespace string) (string, error) {
	pods, err := devtools.ListPods(namespace)
	if err != nil {
		return "", err
	}
	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found in namespace %s", namespace)
	}

	options := make([]huh.Option[string], 0, len(pods))
	for _, p := range pods {
		label := fmt.Sprintf("%s %s (%s)", devtools.StatusIcon(p.Status), p.Name, p.Status)
		options = append(options, huh.NewOption(label, p.Name))
	}

	var selected string
	if err := huh.NewSelect[string]().
		Title("Select pod").
		Options(options...).
		Value(&selected).
		Run(); err != nil {
		return "", nil
	}
	return selected, nil
}

// --- iron dev db ---

func newDevDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database operations",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "connect",
		Short: "Connect to PostgreSQL via psql",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			return connectDB(cfg.Metadata.Name)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "credentials",
		Short: "Show database connection details",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := findProject()
			if err != nil {
				return err
			}
			return printDBCredentials(cfg.Metadata.Name)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "hasura",
		Short: "Open Hasura console in browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openHasuraConsole()
		},
	})

	return cmd
}

func runDBInteractive(projectName string) error {
	var choice string
	if err := huh.NewSelect[string]().
		Title("Database").
		Options(
			huh.NewOption("Connect (psql)", "connect"),
			huh.NewOption("Show credentials", "credentials"),
			huh.NewOption("Open Hasura console", "hasura"),
			huh.NewOption("Back", "back"),
		).
		Value(&choice).
		Run(); err != nil {
		return nil
	}

	switch choice {
	case "connect":
		return connectDB(projectName)
	case "credentials":
		return printDBCredentials(projectName)
	case "hasura":
		return openHasuraConsole()
	}
	return nil
}

func connectDB(dbName string) error {
	tui.NewStatusPrinter().Info("Connecting to PostgreSQL...")
	cmd := exec.Command("psql", "-h", "postgresql", "-p", "5432", "-U", "postgres", "-d", dbName)
	cmd.Env = append(os.Environ(), "PGPASSWORD=postgres")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func printDBCredentials(projectName string) error {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tui.ColorSecondary).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	content.WriteString(tui.BoldStyle.Render("PostgreSQL") + "\n\n")
	fmt.Fprintf(&content, "  Database: %s\n", projectName)
	content.WriteString("  Host:     postgresql\n")
	content.WriteString("  Port:     5432\n")
	content.WriteString("  User:     postgres\n")
	content.WriteString("  Password: postgres\n")
	fmt.Fprintf(&content, "  URL:      postgresql://postgres:postgres@postgresql:5432/%s\n", projectName)

	fmt.Println()
	fmt.Println(boxStyle.Render(content.String()))
	return nil
}

func openHasuraConsole() error {
	printer := tui.NewStatusPrinter()

	// Default local hasura console URL
	url := "http://localhost:8080"

	printer.Info("Opening Hasura console at " + url + "...")

	// Try platform-appropriate open command
	var cmd *exec.Cmd
	switch {
	case isCommandAvailable("open"):
		cmd = exec.Command("open", url)
	case isCommandAvailable("xdg-open"):
		cmd = exec.Command("xdg-open", url)
	default:
		printer.Info("Open in browser: " + url)
		return nil
	}

	return cmd.Run()
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// --- iron dev argocd ---

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

			apps, err := devtools.ListArgoApps()
			if err != nil {
				return err
			}
			if len(apps) == 0 {
				printer.Info("No ArgoCD applications found.")
				return nil
			}

			var toSync []string
			if all {
				for _, app := range apps {
					toSync = append(toSync, app.Name)
				}
			} else {
				// Interactive: pre-check out-of-sync apps
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

				toSync = preSelected
				if err := huh.NewMultiSelect[string]().
					Title("Select applications to sync").
					Options(options...).
					Value(&toSync).
					Run(); err != nil {
					return nil
				}
			}

			if len(toSync) == 0 {
				printer.Info("No applications selected.")
				return nil
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

	syncStyle := tui.SuccessStyle
	if detail.SyncStatus == "OutOfSync" {
		syncStyle = tui.WarningStyle
	}
	healthStyle := tui.SuccessStyle
	switch detail.HealthStatus {
	case "Progressing":
		healthStyle = tui.WarningStyle
	case "Degraded", "Missing":
		healthStyle = tui.ErrorStyle
	}

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
			syncStyle := tui.SuccessStyle
			if app.SyncStatus == "OutOfSync" {
				syncStyle = tui.WarningStyle
			}
			healthStyle := tui.SuccessStyle
			switch app.HealthStatus {
			case "Progressing":
				healthStyle = tui.WarningStyle
			case "Degraded", "Missing":
				healthStyle = tui.ErrorStyle
			}

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

// --- iron dev images ---

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

// --- iron dev gcloud ---

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
