package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthListCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List authenticated accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			results, err := service.AuthAccounts(cmd.Context())
			if err != nil {
				return err
			}
			return output(cmd, results, func(items []app.AuthAccountResult) {
				if len(items) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No accounts.")
					return
				}
				for _, item := range items {
					marker := " "
					if item.Active {
						marker = "*"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s %s  %s  %s\n", marker, item.Handle, item.DID, item.Method)
				}
			})
		},
	}
}
