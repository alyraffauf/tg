package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if auth == nil || !auth.IsAuthenticated() {
			return output(authStatusResult{}, func(_ authStatusResult) {
				fmt.Println("Not logged in.")
			})
		}

		author := resolveAuthor(ctx, auth.CurrentDID().String())
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
