package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search controls by natural-language query",
	Long:  `Search NIST SP 800-53 Rev 5 controls using a natural-language query and print matching controls as JSON.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

var flagSearchLimit int

func init() {
	searchCmd.Flags().IntVar(&flagSearchLimit, "limit", 10, "Maximum number of results (max 50).")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	st, err := buildStore(ctx)
	if err != nil {
		return fmt.Errorf("initialise store: %w", err)
	}
	defer st.Close()

	limit := flagSearchLimit
	if limit > 50 {
		limit = 50
	}

	results, err := st.SearchControls(ctx, args[0], limit)
	if err != nil {
		return err
	}

	return prettyPrint(os.Stdout, results)
}

func prettyPrint(w *os.File, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
