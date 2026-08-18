package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

// initFileLog redirects Go's standard log package output to the startup log
// file so that all log.Println/Printf calls from any package are captured.
func initFileLog() {
	p := startupLogPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.Ltime)
}

// startupLogPath returns the path to the startup error log file.
// %LOCALAPPDATA%\w2e\startup.log
func startupLogPath() string {
	la := os.Getenv("LOCALAPPDATA")
	if la == "" {
		la = os.TempDir()
	}
	return filepath.Join(la, "w2e", "startup.log")
}

func init() {
	// Recover from any panic during startup and write it to a log file so
	// the user can diagnose why w2e.exe won't open.
	// This runs via defer+recover in the goroutine that panics.
}

// logStartup writes a message to the startup log file.
func logStartup(format string, args ...any) {
	p := startupLogPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] %s\n", ts, fmt.Sprintf(format, args...))
}

// logPanic writes a panic's stack trace to the startup log.
func logPanic(r any) {
	logStartup("PANIC: %v\n%s", r, debug.Stack())
}
