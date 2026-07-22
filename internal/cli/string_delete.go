package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newStringDeleteCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <rkey>",
		Short: "Delete a string from your Tangled account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			result, err := service.DeleteString(ctx, args[0])
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.DeletedRecordResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Deleted string %s\n", result.Rkey)
			})
		},
	}
}
