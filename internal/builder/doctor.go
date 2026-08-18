package builder

import (
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"

	"github.com/minfu/w2e/internal/platform"
	w2eruntime "github.com/minfu/w2e/internal/runtime"
)

// DoctorResult is the output schema of `w2e_doctor` (spec §43).
type DoctorResult struct {
	OS              string             `json:"os"`
	Arch            string             `json:"arch"`
	Webview2Runtime bool               `json:"webview2_runtime"`
	Permissions     bool               `json:"permissions"`
	Go              bool               `json:"go"`
	BuildAvailable  bool               `json:"build_available"`
	Webview2Source  string             `json:"webview2_source,omitempty"`
	CrossTargets    []CrossTargetInfo  `json:"cross_targets"`
	Targets         map[string]TargetInfo `json:"targets"`
}

// CrossTargetInfo describes whether building a non-host platform from here
// is known to be possible (best-effort heuristic).
type CrossTargetInfo struct {
	Target   string `json:"target"`
	Cross    bool   `json:"cross"`
	Go       bool   `json:"go"`
	CC       bool   `json:"cc"`
	Notes    string `json:"notes,omitempty"`
}

// TargetInfo adds per-target compile capability to `w2e_doctor`.
type TargetInfo struct {
	Available bool   `json:"available"`
	NeedsCGO  bool   `json:"needs_cgo"`
	Format    string `json:"format"`
	Notes     string `json:"notes,omitempty"`
}

// Doctor reports the host build-environment suitability (§33, §43).
// It is shared by the CLI "doctor" command and the w2e_doctor MCP tool.
func Doctor() DoctorResult {
	d := w2eruntime.Detect()
	res := DoctorResult{
		OS:              stdruntime.GOOS,
		Arch:            stdruntime.GOARCH,
		Webview2Runtime: d.Kind != w2eruntime.KindNotFound,
		Webview2Source:  d.Source,
		Targets:         map[string]TargetInfo{},
		CrossTargets:    []CrossTargetInfo{},
	}
	// Go toolchain reachable?
	if p, err := exec.LookPath("go"); err == nil && p != "" {
		res.Go = true
	}
	// Permissions check (cross-platform). Try writing a sentinel file in a
	// per-OS-local data dir.
	dataDir := localDataDir()
	if dataDir != "" {
		testFile := filepath.Join(dataDir, "w2e", ".w2e-doctor")
		if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err == nil {
			if err := os.WriteFile(testFile, []byte("x"), 0o644); err == nil {
				res.Permissions = true
				_ = os.Remove(testFile)
			}
		}
	}
	// Windows builds use the pre-compiled host template (no Go compiler needed
	// at runtime), so they're always available as long as we can write files.
	res.BuildAvailable = res.Permissions
	if stdruntime.GOOS == "windows" {
		res.BuildAvailable = true // template approach always works
	} else {
		res.BuildAvailable = res.Go && res.Permissions
	}

	// Per-target compile capability via internal/platform.
	if res.Targets == nil {
		res.Targets = map[string]TargetInfo{}
	}
	if res.CrossTargets == nil {
		res.CrossTargets = []CrossTargetInfo{}
	}
	for _, t := range platform.AllTargets {
		info := TargetInfo{
			NeedsCGO: t.NeedsCGO(),
			Format:   t.Format(),
		}
		cross := string(t) != stdruntime.GOOS
		goAvailable := res.Go
		// CGO target needs a C compiler.
		ccAvailable := true
		if t.NeedsCGO() {
			ccName := "gcc"
			if cross {
				switch t {
				case platform.TargetLinux:
					ccName = "x86_64-linux-gnu-gcc"
				case platform.TargetDarwin:
					ccName = "o64-clang"
				}
			} else if stdruntime.GOOS == "darwin" {
				ccName = "clang"
			}
			if _, err := exec.LookPath(ccName); err != nil {
				ccAvailable = false
			}
		}
		// Windows uses the pre-compiled host template — always available.
		if t == platform.TargetWindows {
			info.Available = true
		} else {
			info.Available = goAvailable && (!t.NeedsCGO() || ccAvailable)
		}
		if !info.Available {
			if t.NeedsCGO() && !ccAvailable {
				if cross {
					info.Notes = "missing cross C compiler: install the matching toolchain (see README)"
				} else {
					info.Notes = "missing C compiler: install gcc+libwebkit2gtk-dev (Linux) or Xcode CLT (macOS)"
				}
			}
		}
		res.Targets[string(t)] = info
		res.CrossTargets = append(res.CrossTargets, CrossTargetInfo{
			Target: string(t),
			Cross:  cross,
			Go:     goAvailable,
			CC:     ccAvailable,
			Notes:  info.Notes,
		})
	}

	return res
}

// localDataDir returns a platform-appropriate writable data dir.
func localDataDir() string {
	switch stdruntime.GOOS {
	case "windows":
		return os.Getenv("LOCALAPPDATA")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support")
	default: // linux / bsd
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return xdg
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share")
	}
}
