package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthStatusCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.AuthStatus(cmd.Context())
			if err != nil {
				return err
			}
			return output(cmd, result, func(r *app.AuthStatusResult) {
				if !r.Authenticated {
					fmt.Fprintln(cmd.OutOrStdout(), "Not logged in.")
					return
				}
				switch r.Status {
				case app.SessionStatusActive:
					fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s\n", r.Handle)
				case app.SessionStatusExpired:
					fmt.Fprintln(cmd.OutOrStdout(), "Session expired. Run \"tg auth login\" to re-authenticate.")
				case app.SessionStatusUnknown:
					fmt.Fprintln(cmd.OutOrStdout(), "Unable to verify session (network error).")
				}
			})
		},
	}
}
