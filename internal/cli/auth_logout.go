package cli

import (
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthLogoutCommand(service *app.Service) *cobra.Command {
	var logoutAll bool

	command := &cobra.Command{
		Use:   "logout",
		Short: "Log out of an account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.Logout(cmd.Context(), logoutAll)
			if err != nil {
				return err
			}
			return output(cmd, result, func(r *app.AuthLogoutResult) {
				renderAuthLogout(cmd.OutOrStdout(), r)
			})
		},
	}
	command.Flags().BoolVar(&logoutAll, "all", false, "Log out all accounts")
	return command
}

func renderAuthLogout(writer io.Writer, result *app.AuthLogoutResult) {
	styles := newAuthOutputStyles(isTerminal(writer))
	if result.WasLoggedIn {
		writeAuthHeading(writer, styles.active, "✓", "Logged out")
		return
	}
	writeAuthHeading(writer, styles.inactive, "○", "Not logged in")
}
