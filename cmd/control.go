package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var controlCmd = &cobra.Command{
	Use:   "control <id>",
	Short: "Fetch a single control by ID",
	Long:  `Retrieve the full text of a NIST SP 800-53 Rev 5 control and print it as JSON.`,
	Example: `  fisma-ref-mcp control AC-1
  fisma-ref-mcp control si-3
  fisma-ref-mcp control "AC-2(1)"`,
	Args: cobra.ExactArgs(1),
	RunE: runControl,
}

func init() {
	rootCmd.AddCommand(controlCmd)
}

func runControl(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	st, err := buildStore(ctx)
	if err != nil {
		return fmt.Errorf("initialise store: %w", err)
	}
	defer st.Close()

	c, err := st.GetControl(ctx, args[0])
	if err != nil {
		return err
	}

	return prettyPrint(os.Stdout, c)
}
