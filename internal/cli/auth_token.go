package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthTokenCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "token",
		Short: "Print the active account's access token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, err := service.AccessToken(cmd.Context())
			if err != nil {
				return err
			}
			return output(cmd, app.AccessTokenResult{Token: token}, func(result app.AccessTokenResult) {
				fmt.Fprintln(cmd.OutOrStdout(), result.Token)
			})
		},
	}
}
