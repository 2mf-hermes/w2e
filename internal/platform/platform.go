// Package platform defines the cross-platform output targets w2e supports.
//
// w2e can produce three native executable formats from one web project:
//
//   TargetWindows → Windows PE/EXE (GUI subsystem, no console window)
//   TargetLinux   → Linux ELF (single static or dynamic binary)
//   TargetDarwin  → macOS Mach-O (or .app bundle)
//
// The caller of BuildConfig sets Target (or lists several Targets for
// "build all"). When the target differs from the host build platform, cross
// compilation kicks in: Windows→Linux/darwin requires CGO + a cross C
// toolchain; native (build host = target) also requires CGO for Linux/darwin
// because the cross-platform webview bindings depend on it (WebKit/WKWebView).
package platform

import (
	"fmt"
	"runtime"
	"strings"
)

// Target is one output platform's logical identifier.
type Target string

const (
	// TargetWindows produces a Windows PE executable.
	TargetWindows Target = "windows"
	// TargetLinux produces a Linux ELF executable.
	TargetLinux Target = "linux"
	// TargetDarwin produces a macOS Mach-O executable.
	TargetDarwin Target = "darwin"
	// TargetAll is a meta-target that expands to all three real targets.
	TargetAll Target = "all"
)

// AllTargets is the canonical list of real (non-meta) targets, in the order
// they are emitted by "build all".
var AllTargets = []Target{TargetWindows, TargetLinux, TargetDarwin}

// IsValid reports whether t is a recognized target (including the "all" meta).
func IsValid(t Target) bool {
	switch t {
	case TargetWindows, TargetLinux, TargetDarwin, TargetAll:
		return true
	}
	return false
}

// Expand "all" → [windows,linux,darwin]. Other targets pass through.
func Expand(t Target) []Target {
	if t == TargetAll {
		out := make([]Target, len(AllTargets))
		copy(out, AllTargets)
		return out
	}
	return []Target{t}
}

func (t Target) String() string { return string(t) }

// Parse turns a user-supplied string into a Target. Accepts aliases
// (win/win32/macosx/osx/macos) case-insensitively.
func Parse(s string) (Target, error) {
	low := strings.ToLower(strings.TrimSpace(s))
	switch low {
	case "", "all":
		return TargetAll, nil
	case "windows", "win", "win32", "win64":
		return TargetWindows, nil
	case "linux", "gnu", "unix":
		return TargetLinux, nil
	case "macos", "darwin", "osx", "macosx", "mac":
		return TargetDarwin, nil
	}
	return "", fmt.Errorf("unknown build target: %q (want windows, linux, darwin, or all)", s)
}

// GOOS returns the GOOS value used for `go build` / cross compilation.
func (t Target) GOOS() string { return string(t) }

// GOARCH returns the recommended default GOARCH for the target.
//   windows → amd64
//   linux   → amd64
//   darwin  → amd64 (universal2 is a 2-arch slice; we emit amd64 to start)
func (t Target) GOARCH() string {
	switch t {
	case TargetWindows, TargetLinux, TargetDarwin:
		return "amd64"
	}
	return "amd64"
}

// NeedsCGO reports whether compiling for this target requires CGO.
// Windows uses the pure-Go go-webview2 (no CGO). Linux/darwin use the
// cross-platform webview bindings which depend on C toolchains for
// WebKit2GTK / WKWebView.
func (t Target) NeedsCGO() bool {
	switch t {
	case TargetLinux, TargetDarwin:
		return true
	case TargetWindows:
		return false
	}
	return false
}

// Format returns the human-readable executable format name.
func (t Target) Format() string {
	switch t {
	case TargetWindows:
		return "PE (Portable Executable, Windows GUI subsystem)"
	case TargetLinux:
		return "ELF (Executable and Linkable Format)"
	case TargetDarwin:
		return "Mach-O (macOS 64-bit executable)"
	}
	return "unknown"
}

// Suffix returns the file suffix for the target binary: ".exe" for Windows,
// "" for Linux and macOS.
func (t Target) Suffix() string {
	if t == TargetWindows {
		return ".exe"
	}
	return ""
}

// CrossCompile reports whether building Target t from host OS h requires
// cross-compilation.
func CrossCompile(t Target, hostOS string) bool {
	return string(t) != hostOS
}

// Host returns the runtime target corresponding to the host build machine.
func Host() Target {
	switch runtime.GOOS {
	case "linux":
		return TargetLinux
	case "darwin":
		return TargetDarwin
	default:
		return TargetWindows
	}
}
