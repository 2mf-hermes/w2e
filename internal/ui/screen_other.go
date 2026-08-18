//go:build !windows

package ui

// screenMetrics returns the primary monitor's width and height in pixels.
// On non-Windows platforms, returns a sensible default.
func screenMetrics() (w, h int) {
	// TODO: use xrandr (Linux) or CGDisplayBounds (macOS) for real values.
	return 1920, 1080
}
