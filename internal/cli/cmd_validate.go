package cli

import (
	"fmt"
	"io"

	"github.com/minfu/w2e/internal/config"
	"github.com/minfu/w2e/internal/validator"
)

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "validate: missing SOURCE_DIR argument")
		return 2
	}
	entry := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--entry" && i+1 < len(args) {
			entry = args[i+1]
			i++
		}
	}
	cfg := &config.BuildConfig{SourceDir: args[0], EntryFile: entry}
	rep, err := validator.Validate(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "✕ %s\n", err.Error())
		return 1
	}
	fmt.Fprintln(stdout, "Project validation")
	fmt.Fprintln(stdout, "")
	for _, r := range rep.Results {
		mark := "✓"
		switch r.Status {
		case validator.StatusOK:
			mark = "✓"
		case validator.StatusWarn:
			mark = "⚠"
		case validator.StatusFail:
			mark = "✕"
		case validator.StatusSkip:
			mark = "·"
		}
		fmt.Fprintf(stdout, "  %s %-16s %s\n", mark, r.Check, r.Message)
	}
	if rep.Ready {
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "Project is ready for build.")
		return 0
	}
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stderr, "Project validation failed:")
	for _, e := range rep.Errors {
		fmt.Fprintf(stderr, "  - %s\n", e)
	}
	for _, w := range rep.Warnings {
		fmt.Fprintf(stdout, "  warning: %s\n", w)
	}
	return 1
}
