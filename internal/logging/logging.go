// Package logging provides a minimal, dependency-free leveled logger.
//
// The produced EXE has no console window (-H windowsgui), so log output is
// written to rotating files under %LOCALAPPDATA%\w2e\logs. In debug mode the
// logger additionally echoes to os.Stderr for developer visibility (which is
// safe because the GUI subsystem swallows stderr without opening a console,
// but the CLI may surface it via --debug or via w2e --debug).
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level controls logger verbosity.
type Level int

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

// Logger is a leveled, concurrency-safe logger.
type Logger struct {
	mu      sync.Mutex
	file    io.Writer
	stderr  io.Writer
	level   Level
	prefix  string
}

// New creates a logger writing to the given directory. dir="" disables file
// logging (logs go only to the provided sink). If dir is non-empty it must
// be created by the caller; the log file name is timestamped.
func New(dir string, level Level, prefix string) *Logger {
	l := &Logger{level: level, prefix: prefix}
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err == nil {
			name := fmt.Sprintf("w2e-%s.log", time.Now().Format("20060102"))
			if f, err := os.OpenFile(filepath.Join(dir, name),
				os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				l.file = f
			}
		}
	}
	if level >= LevelDebug {
		l.stderr = os.Stderr
	}
	return l
}

// Default logs dir resolves under %LOCALAPPDATA%\w2e\logs by default.
func Default(level Level, prefix string) *Logger {
	return New(filepath.Join(localAppData(), "w2e", "logs"), level, prefix)
}

// SetMin sets the minimum log level.
func (l *Logger) SetMin(level Level) { l.mu.Lock(); l.level = level; l.mu.Unlock() }

func (l *Logger) writef(level Level, tag, format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if level > l.level {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("%s %s [%s] ", ts, l.prefix, tag) + fmt.Sprintf(format, args...) + "\n"
	if l.file != nil {
		_, _ = l.file.Write([]byte(line))
	}
	if l.stderr != nil {
		_, _ = l.stderr.Write([]byte(line))
	}
}

func (l *Logger) Debugf(f string, a ...any) { l.writef(LevelDebug, "DEBUG", f, a...) }
func (l *Logger) Infof(f string, a ...any)  { l.writef(LevelInfo, "INFO", f, a...) }
func (l *Logger) Warnf(f string, a ...any)  { l.writef(LevelWarn, "WARN", f, a...) }
func (l *Logger) Errorf(f string, a ...any) { l.writef(LevelError, "ERROR", f, a...) }

// Close flushes/closes file handles.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if cl, ok := l.file.(io.Closer); ok {
		_ = cl.Close()
	}
	l.file = nil
}

// localAppData returns %LOCALAPPDATA% or a temporary fallback.
func localAppData() string {
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return v
	}
	if v := os.Getenv("APPDATA"); v != "" {
		return v
	}
	if v, err := os.UserHomeDir(); err == nil {
		return filepath.Join(v, "AppData", "Local")
	}
	return filepath.Join(os.TempDir(), "w2e")
}

// LocalAppData is exported so other packages (which build paths) can reuse it.
func LocalAppData() string { return localAppData() }
