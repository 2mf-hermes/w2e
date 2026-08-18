package mcp

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectSystemPaths(t *testing.T) {
	cases := []struct {
		in         string
		wantError  bool
		wantSubstr string
	}{
		{`C:\Windows\System32\evil.exe`, true, "system"},
		{`C:\Program Files\evil.exe`, true, "system"},
		{`C:\Program Files (x86)\evil.exe`, true, "system"},
		{`C:\ProgramData\evil.exe`, true, "system"},
		{`C:\Users\me\out\app.exe`, false, ""},
		{`C:\dev\app.exe`, false, ""},
	}
	for _, c := range cases {
		got := rejectSystemPaths(filepath.Clean(c.in))
		switch {
		case c.wantError && got == nil:
			t.Errorf("rejectSystemPaths(%q): expected error, got nil", c.in)
		case c.wantError && got != nil:
			if env, ok := got.(*errEnvelope); !ok {
				t.Errorf("rejectSystemPaths(%q): expected *errEnvelope, got %T", c.in, got)
			} else if !strings.Contains(env.Message, c.wantSubstr) {
				t.Errorf("rejectSystemPaths(%q): error %q does not contain %q", c.in, env.Message, c.wantSubstr)
			}
		case !c.wantError && got != nil:
			t.Errorf("rejectSystemPaths(%q): expected nil, got %v", c.in, got)
		}
	}
}

func TestNewErrEnvelopeSuggestion(t *testing.T) {
	env := newErrEnvelope("ENTRY_NOT_FOUND", "missing index.html", nil)
	if env.ErrorCode != "ENTRY_NOT_FOUND" {
		t.Errorf("ErrorCode = %q", env.ErrorCode)
	}
	if !strings.Contains(env.Suggestion, "--entry") {
		t.Errorf("expected --entry suggestion, got %q", env.Suggestion)
	}
	envErr := newErrEnvelope("INVALID_CONFIG", "bad", nil)
	if envErr == nil || envErr.ErrorCode != "INVALID_CONFIG" {
		t.Errorf("unexpected envelope: %+v", envErr)
	}
}
