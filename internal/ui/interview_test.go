package ui

import (
	"reflect"
	"testing"

	"github.com/cameronsjo/claunch/internal/launch"
)

func TestModelChoices_StandardAlias_NoDuplication(t *testing.T) {
	for _, alias := range []string{"opus", "sonnet", "haiku"} {
		got := modelChoices(alias)
		want := []string{"opus", "sonnet", "haiku"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("modelChoices(%q) = %v, want %v", alias, got, want)
		}
	}
}

func TestModelChoices_CustomModel_PrependedAndSelectable(t *testing.T) {
	got := modelChoices("claude-opus-4-8")
	want := []string{"claude-opus-4-8", "opus", "sonnet", "haiku"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("modelChoices(custom) = %v, want %v", got, want)
	}
}

func TestModelChoices_Empty_FallsBackToAliases(t *testing.T) {
	got := modelChoices("")
	want := []string{"opus", "sonnet", "haiku"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("modelChoices(\"\") = %v, want %v", got, want)
	}
}

func TestSessionMode(t *testing.T) {
	cases := []struct {
		in   string
		want launch.SessionMode
	}{
		{"new", launch.New},
		{"resume", launch.Resume},
		{"fork", launch.Fork},
		{"anything-else", launch.New},
	}
	for _, tc := range cases {
		if got := sessionMode(tc.in); got != tc.want {
			t.Errorf("sessionMode(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
