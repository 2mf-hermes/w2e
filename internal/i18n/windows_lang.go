//go:build windows

// windows_lang.go provides real Windows UI-language detection for i18n.
// It calls GetUserDefaultLocaleName (kernel32) via syscall — no external
// dependency on golang.org/x/sys is needed for a single API.
package i18n

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocaleName   = kernel32.NewProc("GetUserDefaultLocaleName")
	procGetSystemDefaultLocaleName = kernel32.NewProc("GetSystemDefaultLocaleName")
)

// winDefaultLocaleName returns the Windows user UI locale as a BCP-47-style
// name (e.g. "zh-TW", "en-US", "ja", "ko-KR"). Returns "" on failure.
func winDefaultLocaleName() string {
	if name := callLocaleName(procGetUserDefaultLocaleName); name != "" {
		return name
	}
	return callLocaleName(procGetSystemDefaultLocaleName)
}

// callLocaleName invokes a *LocaleNameW API and decodes the UTF-16 result.
func callLocaleName(p *syscall.LazyProc) string {
	buf := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH
	r, _, _ := p.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return ""
	}
	s := syscall.UTF16ToString(buf)
	// BCP-47 Windows names use "zh-TW", "en-US", etc.
	return s
}

// winDetectWindowsLang is the Windows-only entry that returns a supported
// i18n code ("zh-TW"/"zh-CN"/"en"/"ja"/"ko") from the OS UI language.
func winDetectWindowsLang() string {
	raw := winDefaultLocaleName()
	if raw == "" {
		return ""
	}
	return normalizeLocale(raw)
}
