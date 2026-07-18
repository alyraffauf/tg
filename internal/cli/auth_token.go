package cli

import (
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/spf13/cobra"
)

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the current access token",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		session, err := auth.CurrentSession(ctx)
		if err == nil {
			token, _ := session.GetHostAccessData()
			if token == "" {
				return fmt.Errorf("current session has no access token")
			}
			fmt.Fprintln(cmd.OutOrStdout(), token)
			return nil
		}
		if !errors.Is(err, atproto.ErrNotAuthenticated) {
			return fmt.Errorf("resume OAuth session: %w", err)
		}
		client, _, err := auth.APIClient(ctx)
		if err != nil {
			if errors.Is(err, atproto.ErrNotAuthenticated) {
				return fmt.Errorf("not logged in; run \"tg auth login\" first")
			}
			return fmt.Errorf("resume auth session: %w", err)
		}
		passwordAuth, ok := client.Auth.(*atclient.PasswordAuth)
		if !ok {
			return fmt.Errorf("not logged in; run \"tg auth login\" first")
		}
		token, _ := passwordAuth.GetTokens()
		if token == "" {
			return fmt.Errorf("current session has no access token")
		}
		fmt.Fprintln(cmd.OutOrStdout(), token)
		return nil
	},
}
