package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var familyCmd = &cobra.Command{
	Use:   "family [id]",
	Short: "List control families, or all controls in a specific family",
	Long: `Without an argument, list all control families.
With a family ID, list all base controls in that family.`,
	Example: `  fisma-ref-mcp family
  fisma-ref-mcp family AC
  fisma-ref-mcp family si`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFamily,
}

func init() {
	rootCmd.AddCommand(familyCmd)
}

func runFamily(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	st, err := buildStore(ctx)
	if err != nil {
		return fmt.Errorf("initialise store: %w", err)
	}
	defer st.Close()

	if len(args) == 0 {
		families, err := st.ListFamilies(ctx)
		if err != nil {
			return err
		}
		return prettyPrint(os.Stdout, families)
	}

	controls, err := st.GetFamily(ctx, args[0])
	if err != nil {
		return err
	}
	return prettyPrint(os.Stdout, controls)
}
