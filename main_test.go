package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binPath is the claunch binary built once for the whole integration suite.
var binPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "claunch-it")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(tmp, "claunch")
	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("build claunch: " + err.Error())
	}
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// harness wires up a stub `claude` on PATH that records its argv + a probe env
// var, a project-scoped config matching the (symlink-resolved) cwd, and returns
// a runner. The stub records to outFile; tests read it back.
type harness struct {
	cwd     string
	binDir  string
	outFile string
	env     []string
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	cwd := t.TempDir()
	realCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatalf("evalsymlinks cwd: %v", err)
	}

	binDir := t.TempDir()
	outFile := filepath.Join(t.TempDir(), "claude.out")
	stub := "#!/usr/bin/env bash\n" +
		"{\n" +
		"  echo ARGS_START\n" +
		"  for a in \"$@\"; do echo \"$a\"; done\n" +
		"  echo ARGS_END\n" +
		"  echo \"OTEL_EXPORTER=${OTEL_EXPORTER:-}\"\n" +
		"} > \"$CLAUNCH_TEST_OUT\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "claude"), []byte(stub), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	xdg := t.TempDir()
	confDir := filepath.Join(xdg, "claunch")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir conf: %v", err)
	}
	conf := "" +
		"[defaults]\n" +
		"model = \"opus\"\n" +
		"permission_mode = \"plan\"\n" +
		"allow_danger = true\n\n" +
		"[[project]]\n" +
		"match = \"" + realCwd + "\"\n" +
		"model = \"sonnet\"\n" +
		"env = { OTEL_EXPORTER = \"otlp\" }\n" +
		"add_dir = [\"" + filepath.Join(realCwd, "shared") + "\"]\n"
	if err := os.WriteFile(filepath.Join(confDir, "claunch.conf"), []byte(conf), 0o644); err != nil {
		t.Fatalf("write conf: %v", err)
	}

	env := append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"XDG_CONFIG_HOME="+xdg,
		"CLAUNCH_TEST_OUT="+outFile,
	)
	return &harness{cwd: realCwd, binDir: binDir, outFile: outFile, env: env}
}

func (h *harness) run(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = h.cwd
	cmd.Env = h.env
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("run claunch %v: %v\nstderr: %s", args, err, errb.String())
	}
	return out.String(), errb.String()
}

// recordedArgs parses the argv the stub claude received.
func (h *harness) recordedArgs(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(h.outFile)
	if err != nil {
		t.Fatalf("read stub output: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	var args []string
	in := false
	for _, l := range lines {
		switch l {
		case "ARGS_START":
			in = true
		case "ARGS_END":
			in = false
		default:
			if in {
				args = append(args, l)
			}
		}
	}
	return args
}

func (h *harness) recordedOTEL(t *testing.T) string {
	t.Helper()
	data, _ := os.ReadFile(h.outFile)
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "OTEL_EXPORTER=") {
			return strings.TrimPrefix(l, "OTEL_EXPORTER=")
		}
	}
	return ""
}

func TestIntegration_Builder_AppliesProfileAndPassesThrough(t *testing.T) {
	h := newHarness(t)
	h.run(t, "-p", "hi")

	args := h.recordedArgs(t)
	want := []string{
		"--permission-mode", "plan",
		"--allow-dangerously-skip-permissions",
		"--model", "sonnet",
		"--add-dir", filepath.Join(h.cwd, "shared"),
		"-p", "hi",
	}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Errorf("claude argv =\n  %v\nwant\n  %v", args, want)
	}
	if got := h.recordedOTEL(t); got != "otlp" {
		t.Errorf("OTEL_EXPORTER passed to claude = %q, want %q", got, "otlp")
	}
}

func TestIntegration_AgentsJSON_PurePassthrough(t *testing.T) {
	h := newHarness(t)
	stdout, _ := h.run(t, "agents", "--json")

	args := h.recordedArgs(t)
	want := []string{"agents", "--json"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Errorf("agents --json argv = %v, want %v (no injection)", args, want)
	}
	for _, a := range args {
		if a == "--permission-mode" || a == "--model" {
			t.Errorf("posture leaked into agents --json passthrough: %v", args)
		}
	}
	if stdout != "" {
		t.Errorf("claunch wrote to stdout on agents --json passthrough: %q (must stay byte-clean)", stdout)
	}
}

func TestIntegration_AgentsInteractive_InjectsSubsetAndBannerToStderr(t *testing.T) {
	h := newHarness(t)
	stdout, stderr := h.run(t, "agents", "--cwd", "/x")

	args := h.recordedArgs(t)
	want := []string{
		"agents",
		"--permission-mode", "plan",
		"--allow-dangerously-skip-permissions",
		"--model", "sonnet",
		"--cwd", "/x",
	}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Errorf("agents argv =\n  %v\nwant\n  %v", args, want)
	}
	if !strings.Contains(stderr, "claude agents") {
		t.Errorf("banner not on stderr: %q", stderr)
	}
	if strings.Contains(stdout, "claude agents") {
		t.Errorf("banner leaked to stdout: %q", stdout)
	}
}

func TestIntegration_Which_PrintsResolvedProfile(t *testing.T) {
	h := newHarness(t)
	stdout, _ := h.run(t, "which")

	for _, want := range []string{"sonnet", "plan", h.cwd} {
		if !strings.Contains(stdout, want) {
			t.Errorf("`claunch which` output missing %q\n%s", want, stdout)
		}
	}
}
