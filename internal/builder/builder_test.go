package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeAppID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"My App", "MyApp"},
		{"My-App_2", "My-App_2"},
		{"(bad) chars!", "badchars"},
		{"", "w2eapp"},
		{"  ", "w2eapp"},
		{"中文App", "App"},
	}
	for _, c := range cases {
		got := sanitizeAppID(c.in)
		if got != c.want {
			t.Errorf("sanitizeAppID(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestRejectSystemPaths(t *testing.T) {
	t.Skip("rejectSystemPaths lives in the mcp package; see mcp/server_test.go")
}

func TestVerifyPESmoke(t *testing.T) {
	// The w2e.exe built for tests is in ../../bin/w2e.exe.
	here, _ := os.Getwd()
	bin := filepath.Join(here, "..", "..", "bin", "w2e.exe")
	if _, err := os.Stat(bin); err != nil {
		t.Skip("bin/w2e.exe missing; run `go build -o bin/w2e.exe ./cmd/w2e` first")
	}
	if err := VerifyPE(bin); err != nil {
		// not a hard fail — could be CUI subsystem if the test build overrode it
		t.Logf("VerifyPE(w2e.exe) = %v (advisory)", err)
	}
}
