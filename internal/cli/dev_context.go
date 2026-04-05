package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dag7/ironplate/internal/config"
	"github.com/dag7/ironplate/internal/devtools"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

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

// findContextByEnv searches for a context matching the given environment patterns.
func findContextByEnv(cfg *config.ProjectConfig, namePatterns []string, envNames []string) (string, error) {
	contexts, err := devtools.ListContexts()
	if err != nil {
		return "", err
	}
	// First: match context name against patterns
	for _, ctx := range contexts {
		lower := strings.ToLower(ctx.Name)
		for _, pattern := range namePatterns {
			if strings.Contains(lower, pattern) {
				return ctx.Name, nil
			}
		}
	}
	// Fallback: match via cfg.Spec.Cloud.Environments
	for _, env := range cfg.Spec.Cloud.Environments {
		for _, envName := range envNames {
			if env.Name == envName || env.ShortName == envName {
				for _, ctx := range contexts {
					if strings.Contains(ctx.Name, env.Name) || strings.Contains(ctx.Name, env.ShortName) {
						return ctx.Name, nil
					}
				}
			}
		}
	}
	return "", nil
}

func switchToStagingContext(cfg *config.ProjectConfig) error {
	stagingCtx, err := findContextByEnv(cfg, []string{"staging", "stg"}, []string{"staging", "stg"})
	if err != nil {
		return err
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

	prodCtx, err := findContextByEnv(cfg, []string{"production", "prod", "prd"}, []string{"production", "prd"})
	if err != nil {
		return err
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
