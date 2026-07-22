package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newSSHKeyDeleteCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <rkey>",
		Short: "Delete an SSH key from your Tangled account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			result, err := service.DeleteSSHKey(ctx, args[0])
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.DeletedRecordResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Deleted SSH key %s\n", result.Rkey)
			})
		},
	}
}
