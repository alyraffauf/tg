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

		did, err := auth.CurrentDID(ctx)
		if err != nil {
			if !errors.Is(err, atproto.ErrNotAuthenticated) {
				return fmt.Errorf("resume OAuth session: %w", err)
			}
			return output(authStatusResult{}, func(_ authStatusResult) {
				fmt.Println("Not logged in.")
			})
		}

		author := resolveAuthor(ctx, did.String())
		result := authStatusResult{
			Authenticated: true,
			DID:           author.DID,
			Handle:        author.Handle,
		}
		return output(result, func(status authStatusResult) {
			fmt.Printf("Logged in as %s\n", status.Handle)
		})
	},
}
