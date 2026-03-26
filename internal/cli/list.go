package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/ironplate-dev/ironplate/internal/components"
	"github.com/ironplate-dev/ironplate/internal/config"
	"github.com/ironplate-dev/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available templates, components, and services",
	}

	cmd.AddCommand(newListComponentsCmd())
	cmd.AddCommand(newListServicesCmd())

	return cmd
}

func newListComponentsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "components",
		Short: "List available infrastructure components",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load project config if available (to show installed status).
			var installedComponents map[string]bool
			configPath, err := config.FindConfigFile(".")
			if err == nil {
				cfg, loadErr := config.Load(configPath)
				if loadErr == nil {
					installedComponents = make(map[string]bool)
					for _, c := range cfg.Spec.Infrastructure.Components {
						installedComponents[c] = true
					}
				}
			}

			allComponents := components.All()
			names := components.List()

			fmt.Println()
			fmt.Println(tui.BoldStyle.Render("  Available Components"))
			fmt.Println()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				tui.BoldStyle.Render("NAME"),
				tui.BoldStyle.Render("DESCRIPTION"),
				tui.BoldStyle.Render("TIER"),
				tui.BoldStyle.Render("INSTALLED"),
			)

			for _, name := range names {
				comp := allComponents[name]

				installed := tui.MutedStyle.Render("-")
				if installedComponents != nil && installedComponents[name] {
					installed = tui.CheckMark
				}

				fmt.Fprintf(w, "  %s\t%s\t%d\t%s\n",
					name,
					comp.Description,
					comp.Tier,
					installed,
				)
			}
			w.Flush()
			fmt.Println()

			return nil
		},
	}
}

func newListServicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "services",
		Short: "List services in the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := config.FindConfigFile(".")
			if err != nil {
				return fmt.Errorf("no ironplate project found: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Println()
			fmt.Println(tui.BoldStyle.Render("  Project Services"))
			fmt.Println()

			if len(cfg.Spec.Services) == 0 {
				fmt.Println(tui.MutedStyle.Render("  No services defined."))
				fmt.Println()
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
				tui.BoldStyle.Render("NAME"),
				tui.BoldStyle.Render("TYPE"),
				tui.BoldStyle.Render("GROUP"),
				tui.BoldStyle.Render("PORT"),
			)

			for _, svc := range cfg.Spec.Services {
				group := svc.Group
				if group == "" {
					group = tui.MutedStyle.Render("-")
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%d\n",
					svc.Name,
					svc.Type,
					group,
					svc.Port,
				)
			}
			w.Flush()
			fmt.Println()

			return nil
		},
	}
}
