package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/minfu/w2e/internal/ui"
)

// cmdGUI launches the Liquid-Glass desktop GUI. Equivalent to running `w2e`
// with no subcommand.
func cmdGUI(ctx context.Context) int {
	if err := ui.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "w2e gui:", err)
		return 1
	}
	return 0
}
