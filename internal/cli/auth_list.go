package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List authenticated accounts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		accounts, activeDID, err := auth.Accounts()
		if err != nil {
			return fmt.Errorf("list accounts: %w", err)
		}
		results := make([]authAccountResult, 0, len(accounts))
		for _, account := range accounts {
			handle := account.Handle
			resolved := resolveAuthor(cmd.Context(), account.DID)
			if resolved.Handle != account.DID {
				handle = resolved.Handle
			}
			results = append(results, authAccountResult{
				Active: account.DID == activeDID,
				DID:    account.DID, Handle: handle, Method: account.Method,
			})
		}
		return output(results, func(items []authAccountResult) {
			if len(items) == 0 {
				fmt.Println("No accounts.")
				return
			}
			for _, item := range items {
				marker := " "
				if item.Active {
					marker = "*"
				}
				fmt.Printf("%s %s  %s  %s\n", marker, item.Handle, item.DID, item.Method)
			}
		})
	},
}
