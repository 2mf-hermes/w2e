//go:build !windows

// windows_lang_stub.go provides an empty stub for winDetectWindowsLang on
// non-Windows platforms so defaultForWindows() in i18n.go compiles cleanly.
package i18n

// winDetectWindowsLang is a no-op on non-Windows; the caller falls back to
// "en" (Unix paths go through LANGUAGE/LC_* env vars first).
func winDetectWindowsLang() string { return "" }
