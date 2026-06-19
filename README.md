# claunch

**A per-project launcher for [Claude Code](https://github.com/anthropics/claude-code).**

`claunch` detects the project from your working directory, applies a per-project profile
(model, permission mode, env, extra directories), runs a short guided launch, and then `exec`s
`claude` with the assembled posture. One launcher, configured once, that does the right thing in
every repo.

```
$ cd ~/Projects/minute && claunch
  ┃ Model    › sonnet        (resolved from ~/Projects/minute)
  ┃ Session  › New
→ exec: claude --permission-mode plan --allow-dangerously-skip-permissions --ide \
         --exclude-dynamic-system-prompt-sections --model sonnet \
         --add-dir ~/Projects/minute/shared
```

## Why

Claude Code takes the same launch posture every time — but different projects want different
defaults. A monorepo wants Opus and an extra `--add-dir`; a tiny script repo wants Sonnet and an
OTEL env var. claunch keeps that knowledge in one config file keyed by directory, so you stop
re-typing flags and start in the right posture automatically.

## Install

**Homebrew** (once the tap is published):

```sh
brew install cameronsjo/tap/claunch
```

**Go:**

```sh
go install github.com/cameronsjo/claunch@latest
```

Or grab a binary from [Releases](https://github.com/cameronsjo/claunch/releases). claunch needs
`claude` on your `PATH`.

## Usage

| Invocation | Behavior |
|------------|----------|
| `claunch` | Resolve the profile for the cwd, run the **interview** (Model, New/Resume/Fork), then exec a session. |
| `claunch <claude-args…>` | Resolve + apply the profile, **no interview** — append your args and exec (builder/passthrough). |
| `claunch agents …` | Inject the agents-valid posture subset; **pure passthrough on `--json`/`--help`/`-h`** so scripting stays clean. |
| `claunch which` (or `config`) | Print the resolved profile for the current directory. |
| `claunch init` | Scaffold a starter config at `~/.config/claunch/claunch.conf`. |
| `claunch edit` | Open the config in `$EDITOR`. |
| `claunch doctor` | Check that `claude` is on `PATH` and the config parses. |
| `claunch version` | Print the version. |

A user-supplied flag always wins over the profile — claunch injects posture first and appends
your arguments last (Claude Code is last-flag-wins). So `claunch --model opus` overrides whatever
the profile resolved.

## Configuration

claunch reads `~/.config/claunch/claunch.conf` (respecting `$XDG_CONFIG_HOME`). Scaffold one with
`claunch init`, then inspect the result with `claunch which`.

```toml
[defaults]
model           = "opus"     # claude --model value (alias or full id)
permission_mode = "plan"     # claunch always starts in plan
allow_danger    = true       # adds --allow-dangerously-skip-permissions (reachable, not on)

[[project]]
match           = "~/Projects/minute"   # longest path-prefix of the cwd wins
model           = "sonnet"
env             = { OTEL_EXPORTER = "otlp" }
add_dir         = ["~/Projects/minute/shared"]

[[project]]
match           = "~/Projects/infrastructure"
add_dir         = ["~/Projects/infrastructure/homelab"]
```

**Resolution.** claunch expands `~`, then picks the `[[project]]` whose `match` is the **longest
path-prefix** of the real working directory and merges it over `[defaults]`:

- **scalars** (`model`, `permission_mode`, `allow_danger`) — the project wins when set
- **`env`** — merged; the project wins on key collisions
- **`add_dir`** — concatenated and de-duplicated

No matching project falls back to `[defaults]` alone. With no config file at all, claunch uses
built-in defaults (`opus` / `plan` / danger reachable).

## Launch posture

An interactive session launch assembles:

```
claude --permission-mode <resolved> [--allow-dangerously-skip-permissions] \
       --ide --exclude-dynamic-system-prompt-sections --model <chosen> \
       [--resume | --resume --fork-session] \
       [--add-dir <dir>]…
```

with the profile's `env` merged into the environment. claunch starts in **plan mode** with bypass
**reachable but not on** — toggle into it with Shift+Tab once you're inside Claude Code.

## Development

```sh
make build     # build ./bin/claunch
make test      # go test -race ./...
make check     # golangci-lint run
make snapshot  # goreleaser cross-compile (no publish)
```

Verified against `claude` v2.1.183.

## License

[MIT](LICENSE) © Cameron Sjo
