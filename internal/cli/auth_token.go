package cli

import (
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/spf13/cobra"
)

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the current access token",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if auth == nil || !auth.IsAuthenticated() {
			return fmt.Errorf("not logged in; run \"tg auth login\" first")
		}

		if auth.CurrentDID().String() != "" {
			client, err := auth.APIClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("resume auth session: %w", err)
			}
			if passwordAuth, ok := client.Auth.(*atclient.PasswordAuth); ok {
				token, _ := passwordAuth.GetTokens()
				if token == "" {
					return fmt.Errorf("current session has no access token")
				}
				fmt.Fprintln(cmd.OutOrStdout(), token)
				return nil
			}
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
