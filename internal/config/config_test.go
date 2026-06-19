package config

import (
	"reflect"
	"testing"
)

const testHome = "/home/u"

// resolveAt is a test helper that runs the pure resolution against a fixed home
// and an already-absolute, already-clean cwd, sidestepping the filesystem.
func resolveAt(c *Config, cwd string) Profile {
	return c.resolve(cwd, testHome)
}

func TestResolve_NoProjects_UsesBuiltinDefaults(t *testing.T) {
	c := &Config{}
	got := resolveAt(c, "/home/u/somewhere")

	if got.Model != "opus" {
		t.Errorf("Model = %q, want built-in %q", got.Model, "opus")
	}
	if got.PermissionMode != "plan" {
		t.Errorf("PermissionMode = %q, want built-in %q", got.PermissionMode, "plan")
	}
	if !got.AllowDanger {
		t.Errorf("AllowDanger = false, want built-in true")
	}
	if got.Match != "" {
		t.Errorf("Match = %q, want empty (defaults-only)", got.Match)
	}
}

func TestResolve_DefaultsOverrideBuiltins(t *testing.T) {
	no := false
	c := &Config{Defaults: Defaults{
		Model:          "sonnet",
		PermissionMode: "acceptEdits",
		AllowDanger:    &no,
	}}
	got := resolveAt(c, "/home/u/x")

	if got.Model != "sonnet" {
		t.Errorf("Model = %q, want %q", got.Model, "sonnet")
	}
	if got.PermissionMode != "acceptEdits" {
		t.Errorf("PermissionMode = %q, want %q", got.PermissionMode, "acceptEdits")
	}
	if got.AllowDanger {
		t.Errorf("AllowDanger = true, want explicit false from defaults")
	}
}

func TestResolve_LongestPrefixWins(t *testing.T) {
	c := &Config{Projects: []Project{
		{Match: "~/Projects", Model: "sonnet"},
		{Match: "~/Projects/minute", Model: "haiku"},
	}}
	got := resolveAt(c, "/home/u/Projects/minute/sub")

	if got.Model != "haiku" {
		t.Errorf("Model = %q, want %q (longest prefix wins)", got.Model, "haiku")
	}
	if got.Match != "~/Projects/minute" {
		t.Errorf("Match = %q, want %q", got.Match, "~/Projects/minute")
	}
}

func TestResolve_ExactMatchCountsAsPrefix(t *testing.T) {
	c := &Config{Projects: []Project{{Match: "~/Projects/minute", Model: "haiku"}}}
	got := resolveAt(c, "/home/u/Projects/minute")
	if got.Model != "haiku" {
		t.Errorf("Model = %q, want %q (exact dir should match)", got.Model, "haiku")
	}
}

func TestResolve_ComponentBoundary_NoFalsePrefix(t *testing.T) {
	c := &Config{Projects: []Project{{Match: "~/Projects/minute", Model: "haiku"}}}
	got := resolveAt(c, "/home/u/Projects/minuteworld")
	if got.Match != "" {
		t.Errorf("Match = %q, want empty — %q must not prefix-match minuteworld", got.Match, "~/Projects/minute")
	}
	if got.Model != "opus" {
		t.Errorf("Model = %q, want built-in opus (no project should match)", got.Model)
	}
}

func TestResolve_ScalarMerge_ProjectWinsWhenSet_DefaultsOtherwise(t *testing.T) {
	c := &Config{
		Defaults: Defaults{Model: "opus", PermissionMode: "plan"},
		Projects: []Project{{Match: "~/p", Model: "sonnet"}}, // sets only model
	}
	got := resolveAt(c, "/home/u/p")
	if got.Model != "sonnet" {
		t.Errorf("Model = %q, want project override %q", got.Model, "sonnet")
	}
	if got.PermissionMode != "plan" {
		t.Errorf("PermissionMode = %q, want default %q (project left it unset)", got.PermissionMode, "plan")
	}
}

func TestResolve_AllowDangerOverrideToFalse(t *testing.T) {
	yes, no := true, false
	c := &Config{
		Defaults: Defaults{AllowDanger: &yes},
		Projects: []Project{{Match: "~/p", AllowDanger: &no}},
	}
	got := resolveAt(c, "/home/u/p")
	if got.AllowDanger {
		t.Errorf("AllowDanger = true, want project override to false")
	}
}

func TestResolve_EnvMerge_ProjectWins(t *testing.T) {
	c := &Config{
		Defaults: Defaults{Env: map[string]string{"A": "1", "B": "2"}},
		Projects: []Project{{Match: "~/p", Env: map[string]string{"B": "3", "C": "4"}}},
	}
	got := resolveAt(c, "/home/u/p")
	want := map[string]string{"A": "1", "B": "3", "C": "4"}
	if !reflect.DeepEqual(got.Env, want) {
		t.Errorf("Env = %v, want %v", got.Env, want)
	}
}

func TestResolve_AddDir_ConcatExpandDedupe(t *testing.T) {
	c := &Config{
		Defaults: Defaults{AddDir: []string{"~/a", "~/shared"}},
		Projects: []Project{{Match: "~/p", AddDir: []string{"~/shared", "~/b"}}},
	}
	got := resolveAt(c, "/home/u/p")
	want := []string{"/home/u/a", "/home/u/shared", "/home/u/b"}
	if !reflect.DeepEqual(got.AddDir, want) {
		t.Errorf("AddDir = %v, want %v (concat, ~-expanded, de-duped, order preserved)", got.AddDir, want)
	}
}

func TestResolve_DefaultsOnly_NoMatch_StillExpandsAddDir(t *testing.T) {
	c := &Config{Defaults: Defaults{AddDir: []string{"~/g"}}}
	got := resolveAt(c, "/home/u/elsewhere")
	want := []string{"/home/u/g"}
	if !reflect.DeepEqual(got.AddDir, want) {
		t.Errorf("AddDir = %v, want %v", got.AddDir, want)
	}
}

func TestParse_ExampleConfig(t *testing.T) {
	data := []byte(`
[defaults]
model = "opus"
permission_mode = "plan"
allow_danger = true

[[project]]
match = "~/Projects/minute"
model = "sonnet"
env = { OTEL_EXPORTER = "otlp" }
add_dir = ["~/Projects/minute/shared"]

[[project]]
match = "~/Projects/infrastructure"
add_dir = ["~/Projects/infrastructure/homelab"]
`)
	c, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if c.Defaults.Model != "opus" {
		t.Errorf("Defaults.Model = %q, want %q", c.Defaults.Model, "opus")
	}
	if len(c.Projects) != 2 {
		t.Fatalf("len(Projects) = %d, want 2", len(c.Projects))
	}
	if c.Projects[0].Model != "sonnet" {
		t.Errorf("Projects[0].Model = %q, want %q", c.Projects[0].Model, "sonnet")
	}
	if c.Projects[0].Env["OTEL_EXPORTER"] != "otlp" {
		t.Errorf("Projects[0].Env[OTEL_EXPORTER] = %q, want %q", c.Projects[0].Env["OTEL_EXPORTER"], "otlp")
	}
	if got := c.Projects[1].AddDir; len(got) != 1 || got[0] != "~/Projects/infrastructure/homelab" {
		t.Errorf("Projects[1].AddDir = %v, want one homelab entry", got)
	}
}

func TestExpandTilde(t *testing.T) {
	cases := []struct{ in, want string }{
		{"~", testHome},
		{"~/Projects", "/home/u/Projects"},
		{"/abs/path", "/abs/path"},
		{"relative", "relative"},
		{"~notme/x", "~notme/x"}, // only ~ or ~/ expands
	}
	for _, tc := range cases {
		if got := expandTilde(tc.in, testHome); got != tc.want {
			t.Errorf("expandTilde(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
