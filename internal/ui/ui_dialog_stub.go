//go:build !windows

// ui_dialog_stub.go: non-Windows stubs for the file/folder pickers. The GUI
// build for Linux/macOS uses webview_go (WebKit) which has its own native
// pickers wired through webview bindings; here we return "" so the JS layer
// falls back to manual entry on pure-host shells where the binding isn't
// available.
package ui

func pickDirectory(hwndOwner uintptr) string { return "" }
func pickFile(hwndOwner uintptr) string       { return "" }
