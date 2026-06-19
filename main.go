// Command claunch is a per-project launcher for Claude Code: it resolves a
// profile from the working directory, runs a short guided launch, then execs
// claude with the assembled posture.
package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/cameronsjo/claunch/cmd"
)

// exampleConfig is the starter config used by `claunch init`. Embedding the
// repo-root file keeps it the single source of truth shared with the docs.
//
//go:embed config.example.conf
var exampleConfig string

func main() {
	if err := cmd.Main(os.Args[1:], exampleConfig); err != nil {
		fmt.Fprintln(os.Stderr, "claunch: "+err.Error())
		os.Exit(1)
	}
}
