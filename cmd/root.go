package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/store"
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

// flags shared across all subcommands
var (
	flagEmbeddingProvider string
	flagEmbeddingModel    string
	flagOllamaURL         string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagEmbeddingProvider, "embedding-provider", envOr("EMBEDDING_PROVIDER", ""),
		`Embedding provider for semantic search: "openai" or "ollama". Omit to use SQL fallback.`)
	rootCmd.PersistentFlags().StringVar(&flagEmbeddingModel, "embedding-model", envOr("EMBEDDING_MODEL", ""),
		"Model name for the embedding provider (uses provider default when omitted).")
	rootCmd.PersistentFlags().StringVar(&flagOllamaURL, "ollama-url", envOr("OLLAMA_URL", "http://localhost:11434"),
		"Base URL for the Ollama API.")
}

// Execute is the entrypoint called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildStore initialises the Store using current flag values.
func buildStore(ctx context.Context) (*store.Store, error) {
	return store.New(ctx, store.Config{
		EmbeddingProvider: flagEmbeddingProvider,
		EmbeddingModel:    flagEmbeddingModel,
		OpenAIKey:         os.Getenv("OPENAI_API_KEY"),
		OllamaBaseURL:     flagOllamaURL,
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
