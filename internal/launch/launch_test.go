package launch

import (
	"reflect"
	"testing"

	"github.com/cameronsjo/claunch/internal/config"
)

func TestSessionArgs_New_FullPosture(t *testing.T) {
	p := config.Profile{
		PermissionMode: "plan",
		AllowDanger:    true,
		AddDir:         []string{"/x", "/y"},
	}
	got := SessionArgs(p, "opus", New)
	want := []string{
		"--permission-mode", "plan",
		"--allow-dangerously-skip-permissions",
		"--ide",
		"--exclude-dynamic-system-prompt-sections",
		"--model", "opus",
		"--add-dir", "/x",
		"--add-dir", "/y",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SessionArgs(New) =\n  %v\nwant\n  %v", got, want)
	}
}

func TestSessionArgs_AllowDangerOff_OmitsFlag(t *testing.T) {
	p := config.Profile{PermissionMode: "plan", AllowDanger: false}
	got := SessionArgs(p, "sonnet", New)
	for _, a := range got {
		if a == "--allow-dangerously-skip-permissions" {
			t.Fatalf("flag present despite AllowDanger=false: %v", got)
		}
	}
}

func TestSessionArgs_Resume(t *testing.T) {
	p := config.Profile{PermissionMode: "plan", AllowDanger: true}
	got := SessionArgs(p, "opus", Resume)
	if !containsSeq(got, []string{"--model", "opus", "--resume"}) {
		t.Errorf("Resume posture missing --resume after --model: %v", got)
	}
	if containsStr(got, "--fork-session") {
		t.Errorf("Resume should not include --fork-session: %v", got)
	}
}

func TestSessionArgs_Fork_ImpliesResume(t *testing.T) {
	p := config.Profile{PermissionMode: "plan", AllowDanger: true}
	got := SessionArgs(p, "opus", Fork)
	if !containsSeq(got, []string{"--resume", "--fork-session"}) {
		t.Errorf("Fork posture must be --resume --fork-session: %v", got)
	}
}

func TestBuilderArgs_ProfileFirst_UserArgsLast(t *testing.T) {
	p := config.Profile{PermissionMode: "plan", AllowDanger: true, Model: "sonnet", AddDir: []string{"/s"}}
	got := BuilderArgs(p, []string{"-p", "hi"})
	want := []string{
		"--permission-mode", "plan",
		"--allow-dangerously-skip-permissions",
		"--model", "sonnet",
		"--add-dir", "/s",
		"-p", "hi",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuilderArgs =\n  %v\nwant\n  %v", got, want)
	}
}

func TestBuilderArgs_NoIdeOrExcludeFlags(t *testing.T) {
	// Builder/passthrough must not inject interactive-only flags that break -p.
	p := config.Profile{PermissionMode: "plan", Model: "opus"}
	got := BuilderArgs(p, []string{"-p", "x"})
	for _, bad := range []string{"--ide", "--exclude-dynamic-system-prompt-sections", "--resume"} {
		if containsStr(got, bad) {
			t.Errorf("builder injected interactive-only flag %q: %v", bad, got)
		}
	}
}

func TestAgentsArgs_InjectsSubsetAfterAgents(t *testing.T) {
	p := config.Profile{PermissionMode: "plan", AllowDanger: true, Model: "opus"}
	got := AgentsArgs(p, []string{"agents", "--cwd", "/proj"})
	want := []string{
		"agents",
		"--permission-mode", "plan",
		"--allow-dangerously-skip-permissions",
		"--model", "opus",
		"--cwd", "/proj",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AgentsArgs =\n  %v\nwant\n  %v", got, want)
	}
}

func TestAgentsArgs_AllowDangerOff_OmitsFlag(t *testing.T) {
	p := config.Profile{PermissionMode: "plan", AllowDanger: false, Model: "opus"}
	got := AgentsArgs(p, []string{"agents"})
	if containsStr(got, "--allow-dangerously-skip-permissions") {
		t.Errorf("agents subset injected danger flag despite AllowDanger=false: %v", got)
	}
}

func TestIsAgentsPassthrough(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"agents", "--json"}, true},
		{[]string{"agents", "--help"}, true},
		{[]string{"agents", "-h"}, true},
		{[]string{"agents", "--all", "--json"}, true},
		{[]string{"agents"}, false},
		{[]string{"agents", "--cwd", "/x"}, false},
	}
	for _, tc := range cases {
		if got := IsAgentsPassthrough(tc.args); got != tc.want {
			t.Errorf("IsAgentsPassthrough(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

func TestMergeEnv_ExtraOverridesAndAppends(t *testing.T) {
	base := []string{"PATH=/bin", "HOME=/h"}
	extra := map[string]string{"FOO": "bar", "HOME": "/override"}
	got := MergeEnv(base, extra)
	want := []string{"PATH=/bin", "FOO=bar", "HOME=/override"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MergeEnv =\n  %v\nwant\n  %v", got, want)
	}
}

func TestMergeEnv_EmptyExtra_ReturnsBase(t *testing.T) {
	base := []string{"PATH=/bin"}
	got := MergeEnv(base, nil)
	if !reflect.DeepEqual(got, base) {
		t.Errorf("MergeEnv with no extra = %v, want %v", got, base)
	}
}

// --- helpers ---

func containsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func containsSeq(haystack, seq []string) bool {
	for i := 0; i+len(seq) <= len(haystack); i++ {
		if reflect.DeepEqual(haystack[i:i+len(seq)], seq) {
			return true
		}
	}
	return false
}
