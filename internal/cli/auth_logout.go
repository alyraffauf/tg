package cli

import (
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your AT Protocol account",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := auth.Logout(cmd.Context())
		wasLoggedIn := true
		if err != nil {
			if errors.Is(err, atproto.ErrNotAuthenticated) {
				wasLoggedIn = false
			} else {
				return err
			}
		}
		return output(authLogoutResult{WasLoggedIn: wasLoggedIn}, func(r authLogoutResult) {
			if r.WasLoggedIn {
				fmt.Println("Logged out.")
			} else {
				fmt.Println("Not logged in.")
			}
		})
	},
}
