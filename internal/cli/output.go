package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

// output dispatches structured data to JSON (when --json is set) or to a
// human-readable renderer.
func output[T any](cmd *cobra.Command, data T, human func(T)) error {
	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	human(data)
	return nil
}
