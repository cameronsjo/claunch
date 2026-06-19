// Package config loads claunch's TOML configuration and resolves a per-project
// profile from the current working directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Defaults holds the base posture applied when no project matches (or as the
// floor that a matching project overrides).
type Defaults struct {
	Model          string            `toml:"model"`
	PermissionMode string            `toml:"permission_mode"`
	AllowDanger    *bool             `toml:"allow_danger"`
	Env            map[string]string `toml:"env"`
	AddDir         []string          `toml:"add_dir"`
}

// Project is a directory-keyed override block.
type Project struct {
	Match          string            `toml:"match"`
	Model          string            `toml:"model"`
	PermissionMode string            `toml:"permission_mode"`
	AllowDanger    *bool             `toml:"allow_danger"`
	Env            map[string]string `toml:"env"`
	AddDir         []string          `toml:"add_dir"`
}

// Config is the parsed claunch.conf.
type Config struct {
	Defaults Defaults  `toml:"defaults"`
	Projects []Project `toml:"project"`
}

// Profile is the fully resolved posture for one working directory.
type Profile struct {
	Model          string
	PermissionMode string
	AllowDanger    bool
	Env            map[string]string
	AddDir         []string
	Match          string // original `match` of the winning project; "" when defaults-only
}

// Built-in fallbacks applied when a value is set neither by a project nor by
// [defaults] — and the entire posture when no config file exists.
const (
	builtinModel          = "opus"
	builtinPermissionMode = "plan"
	builtinAllowDanger    = true
)

// Parse decodes a claunch.conf from raw TOML bytes.
func Parse(data []byte) (*Config, error) {
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Load reads and parses the config at path. A missing file is not an error —
// it yields the built-in defaults (an empty Config resolves to them).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	return Parse(data)
}

// DefaultPath returns the config location, honoring $XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "claunch", "claunch.conf"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".config", "claunch", "claunch.conf"), nil
}

// Resolve picks the profile for cwd: it resolves symlinks best-effort, makes the
// path absolute, then applies the pure resolution.
func (c *Config) Resolve(cwd string) Profile {
	home, _ := os.UserHomeDir()
	real := cwd
	if r, err := filepath.EvalSymlinks(cwd); err == nil {
		real = r
	}
	if abs, err := filepath.Abs(real); err == nil {
		real = abs
	}
	return c.resolve(filepath.Clean(real), home)
}

// resolve is the pure resolution: fixed home, already-clean absolute cwd. It is
// the risk-bearing core and is exercised directly by the tests.
func (c *Config) resolve(cwd, home string) Profile {
	d := c.Defaults

	p := Profile{
		Model:          firstNonEmpty(d.Model, builtinModel),
		PermissionMode: firstNonEmpty(d.PermissionMode, builtinPermissionMode),
		AllowDanger:    boolOr(d.AllowDanger, builtinAllowDanger),
		Env:            mergeEnv(nil, d.Env),
		AddDir:         expandAll(d.AddDir, home),
	}

	best := -1
	var win *Project
	for i := range c.Projects {
		if c.Projects[i].Match == "" {
			continue
		}
		m := filepath.Clean(expandTilde(c.Projects[i].Match, home))
		if cwd == m || strings.HasPrefix(cwd, m+string(filepath.Separator)) {
			if len(m) > best {
				best = len(m)
				win = &c.Projects[i]
			}
		}
	}

	if win != nil {
		if win.Model != "" {
			p.Model = win.Model
		}
		if win.PermissionMode != "" {
			p.PermissionMode = win.PermissionMode
		}
		if win.AllowDanger != nil {
			p.AllowDanger = *win.AllowDanger
		}
		p.Env = mergeEnv(p.Env, win.Env)
		p.AddDir = dedupe(append(p.AddDir, expandAll(win.AddDir, home)...))
		p.Match = win.Match
	}

	return p
}

// expandTilde expands a leading ~ or ~/ to the home directory. A bare "~user"
// form is left untouched (claunch does not resolve other users' homes).
func expandTilde(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// expandAll tilde-expands and cleans each entry.
func expandAll(dirs []string, home string) []string {
	if len(dirs) == 0 {
		return nil
	}
	out := make([]string, len(dirs))
	for i, d := range dirs {
		out[i] = filepath.Clean(expandTilde(d, home))
	}
	return out
}

// dedupe drops repeats while preserving first-seen order.
func dedupe(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := in[:0]
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// mergeEnv returns base with over layered on top (over wins on collision).
func mergeEnv(base, over map[string]string) map[string]string {
	if len(base) == 0 && len(over) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func boolOr(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

// SortedEnvKeys returns the profile env keys in deterministic order, for display
// and for stable argv/env assembly.
func SortedEnvKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
