package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/cameronsjo/claunch/internal/config"
	"github.com/cameronsjo/claunch/internal/launch"
)

var (
	okMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	warnMark = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("!")
	failMark = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("✗")
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check claude availability and config validity",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			healthy := true

			if p, err := launch.ClaudePath(); err == nil {
				fmt.Fprintf(out, "%s claude found: %s\n", okMark, p)
			} else {
				healthy = false
				fmt.Fprintf(out, "%s claude not found on PATH\n", failMark)
			}

			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			switch _, statErr := os.Stat(path); {
			case statErr != nil:
				fmt.Fprintf(out, "%s no config at %s — using built-in defaults (run `claunch init`)\n", warnMark, path)
			default:
				if _, err := config.Load(path); err != nil {
					healthy = false
					fmt.Fprintf(out, "%s config at %s failed to parse: %v\n", failMark, path, err)
				} else {
					fmt.Fprintf(out, "%s config valid: %s\n", okMark, path)
				}
			}

			if !healthy {
				return fmt.Errorf("doctor found problems")
			}
			return nil
		},
	}
}
