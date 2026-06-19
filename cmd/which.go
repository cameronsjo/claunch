package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/cameronsjo/claunch/internal/config"
)

var (
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(14)
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("110"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
)

func newWhichCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "which",
		Aliases: []string{"config"},
		Short:   "Print the resolved profile for the current directory",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("determine working directory: %w", err)
			}
			path, _ := config.DefaultPath()
			printProfile(cmd.OutOrStdout(), cfg.Resolve(cwd), cwd, path)
			return nil
		},
	}
}

func printProfile(w io.Writer, p config.Profile, cwd, confPath string) {
	row := func(label, value string) {
		fmt.Fprintln(w, labelStyle.Render(label)+valueStyle.Render(value))
	}

	fmt.Fprintln(w, titleStyle.Render("claunch profile")+dimStyle.Render("  "+cwd))

	if _, err := os.Stat(confPath); err != nil {
		row("config", confPath+"  "+dimStyle.Render("(missing — built-in defaults)"))
	} else {
		row("config", confPath)
	}

	matched := p.Match
	if matched == "" {
		matched = dimStyle.Render("(defaults only)")
	}
	row("matched", matched)
	row("model", p.Model)
	row("permission", p.PermissionMode)
	row("allow danger", fmt.Sprintf("%t", p.AllowDanger))

	if len(p.Env) > 0 {
		keys := config.SortedEnvKeys(p.Env)
		pairs := make([]string, len(keys))
		for i, k := range keys {
			pairs[i] = k + "=" + p.Env[k]
		}
		row("env", strings.Join(pairs, "  "))
	}
	for i, d := range p.AddDir {
		label := ""
		if i == 0 {
			label = "add-dir"
		}
		row(label, d)
	}
}
