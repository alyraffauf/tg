package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func renderRecordWarnings(writer io.Writer, warnings []app.RecordWarning) {
	for _, warning := range warnings {
		if warning.URI == "" {
			fmt.Fprintf(writer, "warning: %s\n", warning.Error)
			continue
		}
		fmt.Fprintf(writer, "warning: %s: %s\n", warning.URI, warning.Error)
	}
}

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
