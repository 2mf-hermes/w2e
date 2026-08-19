// Package runtime detects the WebView2 Runtime on the host.
//
// IMPORTANT (spec §5, §13): WebView2Loader.dll is NOT the same thing as the
// WebView2 Runtime. go-webview2 ships with an embedded WebView2Loader.dll (via
// go-winloader) and uses COM to talk to WebView2; the Runtime itself is the
// per-machine Edge WebView2 component identified by the
//
//	{F3017226-FE2A-4295-8BDF-00C3A9A5E36C}
//
// EdgeUpdate client GUID, or by an app-local "fixed version" bundle unpacked
// next to the EXE.
//
// Detection order (preferred-first):
//  1. System WebView2 Runtime (Evergreen, HKLM then HKCU +1751b1d8ce6432c1d).
//  2. App-local fixed-version bundle in ./runtime next to the EXE.
//
// If neither exists, the produced EXE must still not elevate. The UI/CLI builds
// will check via doctor and report; the produced runtime will surface a clear
// "WebView2 Runtime not found" error UI.
package runtime

// RuntimeKind classifies where the WebView2 Runtime is sourced from.
type RuntimeKind string

const (
	KindSystem      RuntimeKind = "system"       // Evergreen runtime
	KindFixedBundle RuntimeKind = "fixed-bundle" // app-local bundle
	KindNotFound    RuntimeKind = "not-found"
)

// Detection describes what Detect() found.
type Detection struct {
	Kind    RuntimeKind // "system" | "fixed-bundle" | "not-found"
	Version string      // reported runtime version (e.g. 130.0.2849.46)
	Path    string      // runtime base path when known
	Source  string      // short human-readable source description
}
