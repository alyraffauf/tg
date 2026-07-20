package cli

import (
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		status, did, err := auth.SessionStatus(ctx)
		if err != nil {
			if !errors.Is(err, atproto.ErrNotAuthenticated) {
				return fmt.Errorf("check session: %w", err)
			}
			return output(authStatusResult{}, func(_ authStatusResult) {
				fmt.Println("Not logged in.")
			})
		}

		author := resolveAuthor(ctx, did.String())
		result := authStatusResult{
			Authenticated: true,
			Status:        status,
			DID:           author.DID,
			Handle:        author.Handle,
		}
		return output(result, func(r authStatusResult) {
			switch r.Status {
			case atproto.SessionStatusActive:
				fmt.Printf("Logged in as %s\n", r.Handle)
			case atproto.SessionStatusExpired:
				fmt.Println("Session expired. Run \"tg auth login\" to re-authenticate.")
			case atproto.SessionStatusUnknown:
				fmt.Println("Unable to verify session (network error).")
			}
		})
	},
}
