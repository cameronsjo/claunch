# claunch

A per-project launcher for [Claude Code](https://github.com/anthropics/claude-code). Resolves
a profile from the current directory, runs a short guided launch, then `exec`s `claude` with the
assembled posture. Written in Go with [Charm](https://charm.sh) (`huh` + `lipgloss`).

## Commands

```
make build     # build ./bin/claunch
make test      # go test -race ./...
make check     # golangci-lint run
make fix       # golangci-lint run --fix
make snapshot  # goreleaser snapshot cross-compile (no publish)
```

## Layout

| Path | Responsibility |
|------|----------------|
| `main.go` | Thin entry — hands argv to `cmd.Main`. |
| `cmd/` | cobra command tree (`config`/`which`, `edit`, `init`, `doctor`, `version`) **and** the launch orchestration (resolve → interview → exec). |
| `internal/config/` | TOML parse, `~` expansion, longest-prefix cwd→profile resolution, merge. The risk-bearing core — covered by tests. |
| `internal/launch/` | `claude` argv assembly (session / builder / agents posture), env merge, `syscall.Exec`. |
| `internal/ui/` | `huh` interview (model select, New/Resume/Fork). |

## Design invariants

- **`syscall.Exec`, not a subprocess** — claunch replaces itself with `claude` so Ctrl-C, the
  TTY, and exit codes pass through untouched.
- **Injected posture first, user args last** — a user-supplied flag (e.g. `--model`) overrides
  the profile because Claude Code is last-flag-wins.
- **`agents` is special** — inject only the agents-valid subset; on `--json`/`--help`/`-h` it is
  pure passthrough (no banner on stdout) so `claunch agents --json | jq` stays byte-clean.
- **Verified against** `claude` v2.1.183. Flag spellings (`--exclude-dynamic-system-prompt-sections`,
  `--fork-session`, `--ide`, `--add-dir`, `agents --json`) are confirmed against that binary.

## Config

`~/.config/claunch/claunch.conf` (or `$XDG_CONFIG_HOME`). See `config.example.conf`. Resolution:
expand `~`, pick the `[[project]]` whose `match` is the longest path-prefix of the real cwd, merge
over `[defaults]` (scalars: project wins; `env`: merge, project wins; `add_dir`: concat + dedupe).
