// Package cmd is claunch's command tree (cobra) plus the launch orchestration:
// resolve the profile for the cwd, optionally run the interview, then exec claude.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/cameronsjo/claunch/internal/config"
	"github.com/cameronsjo/claunch/internal/launch"
	"github.com/cameronsjo/claunch/internal/ui"
)

// version is overridden at build time via -ldflags -X .../cmd.version=…
var version = "dev"

// exampleConf holds the starter config body (embedded by package main and
// handed to Main) so `claunch init` and config.example.conf share one source.
var exampleConf string

// ownCommands are the args[0] tokens claunch handles itself rather than passing
// through to claude.
var ownCommands = map[string]bool{
	"which": true, "config": true,
	"edit": true, "init": true, "doctor": true,
	"version": true, "--version": true,
	"help": true, "--help": true, "-h": true,
	"completion": true,
}

// Main is the entry point. It routes claunch-own subcommands through cobra and
// everything else into the launcher (which execs claude in place).
func Main(args []string, exampleConfig string) error {
	exampleConf = exampleConfig
	if len(args) > 0 && ownCommands[args[0]] {
		root := newRootCmd()
		root.SetArgs(args)
		return root.Execute()
	}
	return runLaunch(args)
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "claunch [claude args…]",
		Short: "Per-project launcher for Claude Code",
		Long: `claunch resolves a per-project profile from your working directory,
runs a short guided launch, then execs claude.

  claunch                run the interactive launcher (Model, New/Resume/Fork)
  claunch <claude args…> apply the profile and pass your args straight through
  claunch agents …       inject the agents posture; pure passthrough on --json

Run "claunch which" to see the profile resolved for the current directory.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newWhichCmd(), newEditCmd(), newInitCmd(), newDoctorCmd(), newVersionCmd())
	return root
}

// runLaunch resolves the profile and execs claude under the right posture.
func runLaunch(args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determine working directory: %w", err)
	}
	profile := cfg.Resolve(cwd)

	claudePath, err := launch.ClaudePath()
	if err != nil {
		return err
	}

	var claudeArgs []string
	switch {
	case len(args) == 0:
		choice := ui.Choice{Model: profile.Model, Mode: launch.New}
		if isInteractiveTTY() {
			choice, err = ui.Interview(profile)
			if err != nil {
				return err
			}
		}
		claudeArgs = launch.SessionArgs(profile, choice.Model, choice.Mode)
	case args[0] == "agents":
		if launch.IsAgentsPassthrough(args) {
			claudeArgs = args // byte-clean: no injection, no banner
		} else {
			claudeArgs = launch.AgentsArgs(profile, args)
			launch.Banner(os.Stderr, claudeArgs)
		}
	default:
		claudeArgs = launch.BuilderArgs(profile, args)
	}

	env := launch.MergeEnv(os.Environ(), profile.Env)
	return launch.Exec(claudePath, claudeArgs, env)
}

func loadConfig() (*config.Config, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	return config.Load(path)
}

func isInteractiveTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
