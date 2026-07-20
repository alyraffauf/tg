package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authSwitchCmd = &cobra.Command{
	Use:   "switch <handle-or-did>",
	Short: "Select the active account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		account, err := auth.SelectAccount(args[0])
		if err != nil {
			return fmt.Errorf("select account %q: %w", args[0], err)
		}
		resolved := resolveAuthor(cmd.Context(), account.DID)
		return output(authAccountResult{
			Active: true, DID: account.DID, Handle: resolved.Handle, Method: account.Method,
		}, func(item authAccountResult) {
			fmt.Printf("Switched to %s\n", item.Handle)
		})
	},
}
