package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the current OAuth access token",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		session, err := requireAuthSession(cmd.Context())
		if err != nil {
			return err
		}
		token, _ := session.GetHostAccessData()
		if token == "" {
			return fmt.Errorf("current OAuth session has no access token")
		}
		fmt.Fprintln(cmd.OutOrStdout(), token)
		return nil
	},
}
