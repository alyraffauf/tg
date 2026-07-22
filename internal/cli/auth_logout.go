package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthLogoutCommand(service *app.Service) *cobra.Command {
	var logoutAll bool

	command := &cobra.Command{
		Use:   "logout",
		Short: "Log out of your AT Protocol account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Logout(cmd.Context(), logoutAll)
			if err != nil {
				return err
			}
			return output(cmd, result, func(r *app.AuthLogoutResult) {
				if r.WasLoggedIn {
					fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "Not logged in.")
				}
			})
		},
	}
	command.Flags().BoolVar(&logoutAll, "all", false, "Log out all accounts")
	return command
}
