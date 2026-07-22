package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthSwitchCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <handle-or-did>",
		Short: "Select the active account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.SwitchAccount(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output(cmd, result, func(item *app.AuthAccountResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Switched to %s\n", item.Handle)
			})
		},
	}
}
