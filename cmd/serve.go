package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	fismamc "github.com/forgant-foundry/fisma-ref-mcp/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the FISMA reference MCP server.

HTTP mode (default):
  fisma-ref-mcp serve --port 8080

Stdio mode (for Claude Desktop and other MCP clients that use stdio):
  fisma-ref-mcp serve --stdio`,
	RunE: runServe,
}

var (
	flagPort  int
	flagStdio bool
)

func init() {
	serveCmd.Flags().IntVar(&flagPort, "port", 8080, "Port for the HTTP MCP server.")
	serveCmd.Flags().BoolVar(&flagStdio, "stdio", false, "Use stdio transport instead of HTTP.")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := buildStore(ctx)
	if err != nil {
		return fmt.Errorf("initialise store: %w", err)
	}
	defer st.Close()

	s := fismamc.NewServer(st)

	if flagStdio {
		return fismamc.ServeStdio(s)
	}

	addr := fmt.Sprintf(":%d", flagPort)
	fmt.Fprintf(os.Stderr, "fisma-ref-mcp listening on %s\n", addr)
	return fismamc.ServeHTTP(ctx, s, addr)
}
