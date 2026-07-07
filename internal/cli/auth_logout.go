package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your AT Protocol account",
	RunE: func(cmd *cobra.Command, args []string) error {
		if auth == nil {
			return fmt.Errorf("auth is not available")
		}
		if err := auth.Logout(cmd.Context()); err != nil {
			return err
		}
		fmt.Println("Logged out.")
		return nil
	},
}
