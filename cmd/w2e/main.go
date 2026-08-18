// Package main is the w2e command entry point.
//
// It dispatches between subcommands (build, validate, doctor, mcp, version,
// gui) and falls back to launching the GUI when no subcommand is given.
//
// On Windows, w2e.exe is linked with -H windowsgui (subsystem=GUI) so double-
// clicking it never spawns a DOS console window. When a CLI subcommand is
// invoked from a terminal, attachParentConsole() bridges this process to the
// caller's console so stdout/stderr flow back; in GUI mode the attach call
// harmlessly fails and the WebView2 window opens. See console_windows.go.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/minfu/w2e/internal/cli"
)

func main() {
	// In GUI mode (-H windowsgui) stderr is NUL, so redirect Go's log package
	// output to our startup log so UI messages are captured.
	initFileLog()
	defer func() {
		if r := recover(); r != nil {
			logPanic(r)
			fmt.Fprintf(os.Stderr, "w2e crashed: %v\n", r)
			freeConsoleIfAttached()
			os.Exit(1)
		}
	}()

	ctx := context.Background()

	// Sanity check: if a Windows terminal is calling us for a CLI run,
	// wire ourselves into the parent console AFTER all the subsystem-level
	// startup has settled.
	_ = attachParentConsole()

	logStartup("starting w2e (args=%v, pid=%d)", os.Args, os.Getpid())

	code := cli.Run(ctx, os.Args[1:])

	logStartup("cli.Run returned code=%d", code)

	switch code {
	case 0:
		_ = os.Stdout.Sync()
		_ = os.Stderr.Sync()
		freeConsoleIfAttached()
		return
	default:
		_ = os.Stdout.Sync()
		_ = os.Stderr.Sync()
		freeConsoleIfAttached()
		fmt.Fprintf(os.Stderr, "exit code: %d\n", code)
		os.Exit(code)
	}
}
