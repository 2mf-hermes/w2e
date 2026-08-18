// Package cli implements the w2e command-line interface. It wires commands
// (build, validate, doctor, version, mcp, gui) to the shared BuildEngine.
//
// This file is the command dispatcher; per-command implementations live
// alongside in cmd_*.go.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/minfu/w2e/internal/builder"
	"github.com/minfu/w2e/internal/i18n"
	"github.com/minfu/w2e/internal/ui"
	"github.com/minfu/w2e/internal/version"
)

// Run parses os.Args[1:] and executes the matching command. It returns the
// process exit code so tests can assert on it without os.Exit().
func Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		// No args → launch the GUI.
		if err := ui.Run(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "w2e gui:", err)
			return 1
		}
		return 0
	}
	switch args[0] {
	case "help", "-h", "--help":
		printHelp(os.Stdout)
		return 0
	case "version", "--version", "-v":
		return cmdVersion(os.Stdout)
	case "build":
		return cmdBuild(ctx, args[1:], os.Stdout, os.Stderr)
	case "validate":
		return cmdValidate(args[1:], os.Stdout, os.Stderr)
	case "doctor":
		return cmdDoctor(os.Stdout)
	case "mcp":
		return cmdMCP(ctx, args[1:], os.Stdin, os.Stdout, os.Stderr)
	case "gui":
		return cmdGUI(ctx)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", args[0])
		printHelp(os.Stderr)
		return 2
	}
}
// NewEngine returns a configured Builder engine.
func NewEngine() *builder.Engine { return builder.New() }

// DefaultBundle returns the process-wide i18n bundle.
func DefaultBundle() *i18n.Bundle { return i18n.Default() }

// printHelp lists available subcommands.
func printHelp(w *os.File) {
	fmt.Fprintln(w, `w2e — Web Application → Native Executable (Windows / Linux / macOS)

Usage:
  w2e                      Launch the graphical packager
  w2e build SOURCE [flags] Build a native executable from a web project
  w2e validate SOURCE      Validate a web project before building
  w2e doctor               Diagnose the local Build Environment
  w2e mcp                  Start the MCP server (stdio transport)
  w2e version              Print version information
  w2e help                 Show this help

Build flags:
  --entry FILE             Override the entry HTML file (defaults to detected)
  --name NAME              Application name (defaults to output basename)
  --title TITLE            Window title (default: "My App")
  --width N                Initial window width  (default: 1024, min: 320)
  --height N               Initial window height (default: 720, min: 240)
  --icon PATH              Optional .ico or .png icon path (Windows)
  --target PLATFORM        Target: windows, linux, darwin, or all (default: windows)
  --no-resizable           Disable window resize
  --output PATH            Output executable path (required)
  --keep-temp              Keep the temp build dir on failure (for debugging)
  --url                    Build with an online URL instead of SOURCE_DIR

Examples:
  w2e build ./dist --output ./release/MyApp.exe
  w2e build ./dist --target all --output ./release/MyApp
  w2e build ./dist --title "My App" --width 1280 --height 800 --output ./out
  w2e build --url https://example.com --output ./out.exe

See README.md for full documentation.`)
}

// versionString comes from internal/version.
func versionString() string { return version.String() }
