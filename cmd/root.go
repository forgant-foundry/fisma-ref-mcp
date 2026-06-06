package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/rel_store"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fisma-ref-mcp",
	Short: "NIST SP 800-53 Rev 5 reference MCP server",
	Long: `fisma-ref-mcp provides semantic and deterministic access to NIST SP 800-53
Rev 5 security controls via the Model Context Protocol (MCP).

Run as a long-lived server:
  fisma-ref-mcp serve

Or invoke a single tool and receive JSON on stdout:
  fisma-ref-mcp search "multi-factor authentication"
  fisma-ref-mcp control AC-1
  fisma-ref-mcp family AC`,
}

func init() {}

// Execute is the entrypoint called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildStore initialises the Store. Embedding provider and model are
// auto-detected from the pre-built index embedded in the binary; the only
// runtime input is OPENAI_API_KEY (openai variants) or OLLAMA_URL (ollama
// variants, defaults to http://localhost:11434).
func buildStore(ctx context.Context) (*rel_store.Store, error) {
	return rel_store.New(ctx, rel_store.Config{
		OpenAIKey:     os.Getenv("OPENAI_API_KEY"),
		OllamaBaseURL: envOr("OLLAMA_URL", "http://localhost:11434"),
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
