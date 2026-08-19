//go:build !windows

package runtime

// Detect on non-Windows platforms always reports "not-found" — WebView2 is
// a Windows-only component. On macOS/Linux the GUI uses WKWebView/WebKit2GTK
// which are platform-native and always available.
func Detect() Detection {
	return Detection{Kind: KindNotFound, Source: "not applicable (non-Windows)"}
}
