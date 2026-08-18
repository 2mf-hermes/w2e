// cmd/w2e-mcp is a dedicated launcher that starts the w2e MCP server on
// stdio (spec §39). Some agents prefer a dedicated subcommand/binary; this
// is functionally identical to `w2e mcp`.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/minfu/w2e/mcp"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintln(os.Stderr, "w2e-mcp: starting MCP stdio server")
	if err := mcp.Stdio(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		os.Exit(1)
	}
}
