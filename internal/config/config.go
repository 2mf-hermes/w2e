// Package config holds the shared build configuration schema.
//
// GUI, CLI, and MCP all construct a BuildConfig and hand it to the same
// BuildEngine. The schema intentionally stays minimal and serializable so it
// can be round-tripped through MCP JSON without loss.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildConfig is the complete description of one packaging job.
type BuildConfig struct {
	// SourceDir is the directory containing the web project to be embedded.
	SourceDir string `json:"source_dir"`
	// EntryFile is the HTML entry (often index.html). If empty, detected.
	EntryFile string `json:"entry,omitempty"`
	// OutputFile is the resulting .exe path.
	OutputFile string `json:"output"`
	// AppName is used for the embedded module name and resource identity.
	AppName string `json:"app_name,omitempty"`
	// WindowTitle is the native window title of the produced EXE.
	WindowTitle string `json:"window_title,omitempty"`
	// Width / Height are initial window dimensions in pixels.
	Width  int  `json:"width,omitempty"`
	Height int  `json:"height,omitempty"`
	// Resizable controls whether the produced window can be resized.
	Resizable bool `json:"resizable,omitempty"`
	// IconPath is the path to an .ico or .png used as the EXE icon.
	IconPath string `json:"icon,omitempty"`
	// SourceURL, when non-empty, switches mode to "Remote URL" - the
	// produced EXE loads the given URL directly instead of an embedded app.
	SourceURL string `json:"source_url,omitempty"`
	// Language UI language code override (zh-TW/zh-CN/en/ja/ko). If empty, detected.
	Language string `json:"language,omitempty"`
	// Debug toggles verbose runtime logging in the produced EXE.
	Debug bool `json:"debug,omitempty"`
	// Target is the output platform: "windows", "linux", "darwin", or "all".
	// Empty defaults to TargetWindows for backward compatibility.
	Target string `json:"target,omitempty"`
}

// Defaults applied by Normalize() when the caller leaves fields empty.
const (
	DefaultWidth       = 1024
	DefaultHeight     = 720
	MinWindowWidth    = 540
	MinWindowHeight   = 500
	DefaultWindowTitle = "My App"
	DefaultAppName     = "MyApp"
)

// Aliases used by the UI to keep its field naming consistent with CLI flags.
const (
	MinWidth       = MinWindowWidth
	MinHeight      = MinWindowHeight
)

// ErrInvalidConfig indicates a semantic problem with the config fields.
var ErrInvalidConfig = errors.New("invalid build config")

// Normalize fills defaults and applies a baseline of cross-source validation.
// It does NOT touch the filesystem - that's done by builder.Validate.
func (c *BuildConfig) Normalize() error {
	// Mode selection: either SourceDir OR SourceURL must be present.
	if c.SourceDir == "" && c.SourceURL == "" {
		return fmt.Errorf("%w: source_dir or source_url is required", ErrInvalidConfig)
	}
	if c.SourceDir != "" && c.SourceURL != "" {
		return fmt.Errorf("%w: source_dir and source_url are mutually exclusive", ErrInvalidConfig)
	}

	if c.OutputFile == "" {
		return fmt.Errorf("%w: output is required", ErrInvalidConfig)
	}

	if c.AppName == "" {
		base := strings.TrimSuffix(filepath.Base(c.OutputFile), ".exe")
		if base == "" {
			c.AppName = DefaultAppName
		} else {
			c.AppName = base
		}
	}
	// Sanitize AppName to a Go-import-path-safe segment.
	if !isSafeIdent(c.AppName) {
		c.AppName = DefaultAppName
	}

	if c.WindowTitle == "" {
		c.WindowTitle = DefaultWindowTitle
	}
	if c.Width < MinWindowWidth {
		c.Width = DefaultWidth
	}
	if c.Height < MinWindowHeight {
		c.Height = DefaultHeight
	}

	// Default target to Windows when not specified (preserves the original
	// single-platform behavior).
	if c.Target == "" {
		c.Target = "windows"
	}

	return nil
}

// isSafeIdent returns true when s is a sane single Go identifier/path segment.
func isSafeIdent(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// HasSourceDir returns true when this is a local build (not a remote-URL one).
func (c *BuildConfig) HasSourceDir() bool { return c.SourceDir != "" }

// ResolveOutputAbs returns the absolute, cleaned output path.
func (c *BuildConfig) ResolveOutputAbs() (string, error) {
	abs, err := filepath.Abs(c.OutputFile)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// ResolveSourceAbs returns the absolute source dir.
func (c *BuildConfig) ResolveSourceAbs() (string, error) {
	abs, err := filepath.Abs(c.SourceDir)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// FileExists is a tiny helper used by normalization and validation.
func FileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
