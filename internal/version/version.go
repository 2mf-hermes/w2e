// Package version exposes build-time version information for w2e.
//
// Values are injected via -ldflags "-X github.com/minfu/w2e/internal/version.X=...".
package version

import (
	"fmt"
	"runtime"
)

// Build-time constants. These are overridden via -ldflags at release build.
var (
	Version   = "1.0.0-dev"
	GoVersion = runtime.Version()
	Commit    = "unknown"
	BuildDate = "unknown"
)

// MCPProtocolVersion is the MCP protocol version string advertised to clients.
const MCPProtocolVersion = "2025-06-18"

// WebView2Engine is the fixed UI/runtime engine descriptor for documentation.
const WebView2Engine = "Microsoft WebView2 (Edge runtime, embedded via go-webview2)"

// Info aggregates version information.
type Info struct {
	Version           string `json:"version"`
	GoVersion         string `json:"go_version"`
	Commit            string `json:"commit"`
	BuildDate         string `json:"build_date"`
	OS                string `json:"os"`
	Arch              string `json:"arch"`
	MCPProtocol       string `json:"mcp_protocol"`
	WebView2Engine    string `json:"webview2_engine"`
	BuildEngine       string `json:"build_engine_version"`
}

// Get returns the assembled version Info.
func Get() Info {
	return Info{
		Version:        Version,
		GoVersion:      GoVersion,
		Commit:         Commit,
		BuildDate:      BuildDate,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		MCPProtocol:    MCPProtocolVersion,
		WebView2Engine: WebView2Engine,
		BuildEngine:    "1",
	}
}

// String returns a human-readable one-line version banner.
func String() string {
	return fmt.Sprintf("w2e %s (%s/%s) go %s commit %s built %s",
		Version, runtime.GOOS, runtime.GOARCH, GoVersion, Commit, BuildDate)
}
