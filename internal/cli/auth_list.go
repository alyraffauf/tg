package cli

import (
	"io"

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
				renderAuthAccountList(cmd.OutOrStdout(), items)
			})
		},
	}
}

func renderAuthAccountList(writer io.Writer, items []app.AuthAccountResult) {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		active := ""
		if item.Active {
			active = "✓"
		}
		rows = append(rows, []string{active, item.Handle, item.DID, item.Method})
	}
	renderTable(writer, []string{"ACTIVE", "HANDLE", "DID", "METHOD"}, rows, "No accounts.")
}
