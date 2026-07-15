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
		if auth == nil || !auth.IsAuthenticated() {
			return fmt.Errorf("not logged in; run \"tg auth login\" first")
		}

		session, err := auth.CurrentSession(cmd.Context())
		if err != nil {
			return fmt.Errorf("resume OAuth session: %w", err)
		}
		token, _ := session.GetHostAccessData()
		if token == "" {
			return fmt.Errorf("current OAuth session has no access token")
		}
		fmt.Fprintln(cmd.OutOrStdout(), token)
		return nil
	},
}
