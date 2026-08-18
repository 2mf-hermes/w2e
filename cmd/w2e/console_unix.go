//go:build !windows

// console_unix.go: stub implementations of the Windows-only helpers so
// cmd/w2e/main.go compiles on Linux/macOS. There is no console-window game
// here — Unix processes share their parent's stdout/stderr by default — so
// these are no-ops and `go build` produces a normal terminal binary.
package main

// attachParentConsole bridges os.Stdout/Stderr to the parent terminal on
// Windows. On Unix this is always a no-op.
func attachParentConsole() bool { return false }

// freeConsoleIfAttached is the Windows-only cleanup counterpart. On Unix
// this is a no-op.
func freeConsoleIfAttached() {}
