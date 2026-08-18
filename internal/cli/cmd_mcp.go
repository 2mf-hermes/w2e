package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/minfu/w2e/internal/version"
	"github.com/minfu/w2e/mcp"
)

// cmdMCP starts the MCP stdio server. STDIO is required by spec §39.
func cmdMCP(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// redirect stderr is not possible from inside (server writes stderr directly).
	// For the CLI dispatcher we forward stderr to the parent process stderr.
	_ = stdin
	_ = stdout
	// the MCP server must run on real stdin/stdout (not the test stub), so
	// only use this in production. We forward stderr so it keeps streaming.
	fmt.Fprintln(stderr, "w2e: starting MCP stdio server (protocol "+version.Get().MCPProtocol+")")
	if err := mcp.Stdio(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "mcp server error: %v\n", err)
		return 1
	}
	return 0
}
