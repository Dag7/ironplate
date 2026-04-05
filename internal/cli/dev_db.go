package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dag7/ironplate/internal/tui"
	"github.com/spf13/cobra"
)

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
