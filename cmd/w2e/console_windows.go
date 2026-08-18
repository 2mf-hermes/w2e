//go:build windows

// console_windows.go: w2e.exe is built with -H windowsgui (no console
// window), so double-clicking it never spawns a DOS box. CLI subcommands
// (doctor/build/version/...) need stdout/stderr to reach the parent
// terminal instead. attachParentConsole uses kernel32 AttachConsole +
// GetStdHandle and rewires os.Stdout / os.Stderr to the parent's console.
//
// In GUI mode (no subcommand), there's no parent console and AttachConsole
// fails harmlessly — w2e silently opens its WebView2 window, which is the
// desired behavior.
//
// Only the stable Win32 console surface is used, so no x/sys dependency is
// required. os.NewFile on Windows accepts a raw HANDLE as its uintptr
// argument; subsequent Writes on that file reach the console buffer because
// Go's runtime treats console handles specially (non-pollable, blocking
// WriteFile → WriteConsoleW path).
package main

import (
	"os"
	"syscall"
)

var (
	kern32         = syscall.NewLazyDLL("kernel32.dll")
	pAttachConsole = kern32.NewProc("AttachConsole")
	pGetStdHandle  = kern32.NewProc("GetStdHandle")
	pFreeConsole   = kern32.NewProc("FreeConsole")
)

// attachParentProcess is the constant -1 (ATTACH_PARENT_PROCESS).
const attachParentProcess = ^uint32(0)

// STD_OUTPUT_HANDLE / STD_ERROR_HANDLE (-11 / -12 as win32 DWORDs).
const (
	stdOutputHandle = ^uint32(0) - 10 // -11
	stdErrorHandle  = ^uint32(0) - 11 // -12
)

// attachParentConsole attaches w2e to the parent process's console and
// bridges Go's os.Stdout / os.Stderr to it. Returns true on success
// (including the "already own a console" case); returns false when there is
// no parent console (typical for Explorer-launched GUI).
func attachParentConsole() bool {
	r, _, e := pAttachConsole.Call(uintptr(attachParentProcess))
	if r == 0 {
		if e == syscall.Errno(5) { // ERROR_ACCESS_DENIED — already attached
			return true
		}
		return false
	}
	// Fetch the now-attached console's stdout/stderr handles and swap them
	// into os.Stdout / os.Stderr.
	bridge(uintptr(stdOutputHandle), 1)
	bridge(uintptr(stdErrorHandle), 2)
	return true
}

// bridge acquires the requested console std handle and assigns it to
// os.Stdout (fd=1) or os.Stderr (fd=2).
func bridge(stdHandle, fd uintptr) {
	h, _, _ := pGetStdHandle.Call(stdHandle)
	if h == 0 || h == uintptr(syscall.InvalidHandle) {
		return
	}
	f := os.NewFile(h, "")
	if f == nil {
		return
	}
	switch fd {
	case 1:
		os.Stdout = f
	case 2:
		os.Stderr = f
	}
}

// freeConsoleIfAttached releases the parent console we attached so the
// caller's prompt returns promptly. Best-effort, errors ignored. Safe no-op
// if never attached.
func freeConsoleIfAttached() {
	_, _, _ = pFreeConsole.Call()
}
