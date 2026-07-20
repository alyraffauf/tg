package cli

import (
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var authLogoutAll bool

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your AT Protocol account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if authLogoutAll {
			err = auth.LogoutAll(cmd.Context())
		} else {
			err = auth.Logout(cmd.Context())
		}
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

func init() {
	authLogoutCmd.Flags().BoolVar(&authLogoutAll, "all", false, "Log out all accounts")
}
