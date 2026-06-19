// Package launch assembles the claude argv for each posture (session, builder,
// agents), merges environment, and execs claude in place.
package launch

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"github.com/cameronsjo/claunch/internal/config"
)

// SessionMode selects how an interactive session resumes (or doesn't).
type SessionMode int

const (
	New SessionMode = iota
	Resume
	Fork
)

// SessionArgs builds the full interactive posture: plan-mode default, bypass
// reachable (when allowed), IDE + lean system prompt, the chosen model, the
// resume/fork choice, then each --add-dir.
func SessionArgs(p config.Profile, model string, mode SessionMode) []string {
	args := []string{"--permission-mode", p.PermissionMode}
	if p.AllowDanger {
		args = append(args, "--allow-dangerously-skip-permissions")
	}
	args = append(args, "--ide", "--exclude-dynamic-system-prompt-sections", "--model", model)
	switch mode {
	case Resume:
		args = append(args, "--resume")
	case Fork:
		args = append(args, "--resume", "--fork-session")
	}
	for _, d := range p.AddDir {
		args = append(args, "--add-dir", d)
	}
	return args
}

// BuilderArgs applies the profile's core posture, then appends the user's claude
// args verbatim. Injected flags go first so a user override (e.g. --model) wins
// under Claude Code's last-flag-wins parsing. Interactive-only flags (--ide,
// --exclude-…, --resume) are intentionally omitted — they break -p/--print.
func BuilderArgs(p config.Profile, userArgs []string) []string {
	args := []string{"--permission-mode", p.PermissionMode}
	if p.AllowDanger {
		args = append(args, "--allow-dangerously-skip-permissions")
	}
	args = append(args, "--model", p.Model)
	for _, d := range p.AddDir {
		args = append(args, "--add-dir", d)
	}
	return append(args, userArgs...)
}

// AgentsArgs injects only the agents-valid posture subset between the "agents"
// subcommand and the user's remaining args. agentArgs[0] must be "agents".
func AgentsArgs(p config.Profile, agentArgs []string) []string {
	out := []string{"agents", "--permission-mode", p.PermissionMode}
	if p.AllowDanger {
		out = append(out, "--allow-dangerously-skip-permissions")
	}
	out = append(out, "--model", p.Model)
	return append(out, agentArgs[1:]...)
}

// IsAgentsPassthrough reports whether `claude agents …` is a scripting/help
// invocation that must reach claude byte-clean: no posture injection, no banner.
func IsAgentsPassthrough(agentArgs []string) bool {
	for _, a := range agentArgs[1:] {
		switch a {
		case "--json", "--help", "-h":
			return true
		}
	}
	return false
}

// MergeEnv overlays extra onto base ("KEY=VALUE" entries). Overridden keys are
// dropped from base and re-appended (sorted) so the result is deterministic.
func MergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(extra))
	for _, e := range base {
		k := e
		if i := strings.IndexByte(e, '='); i >= 0 {
			k = e[:i]
		}
		if _, overridden := extra[k]; overridden {
			continue
		}
		out = append(out, e)
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, k+"="+extra[k])
	}
	return out
}

// Banner writes the informational "→ claude …" line. It always goes to stderr so
// it never corrupts piped stdout (e.g. `claunch agents --json | jq`).
func Banner(w io.Writer, args []string) {
	fmt.Fprintln(w, "→ claude "+strings.Join(args, " "))
}

// ClaudePath resolves the claude binary on PATH.
func ClaudePath() (string, error) {
	p, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found on PATH: %w", err)
	}
	return p, nil
}

// Exec replaces the current process with claude. On success it never returns, so
// Ctrl-C, the TTY, and the exit code pass through untouched.
func Exec(claudePath string, args, env []string) error {
	argv := append([]string{claudePath}, args...)
	return syscall.Exec(claudePath, argv, env)
}
