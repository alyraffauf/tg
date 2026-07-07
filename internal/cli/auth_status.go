package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if auth == nil || !auth.IsAuthenticated() {
			fmt.Println("Not logged in.")
			return nil
		}
		fmt.Printf("Logged in as %s\n", auth.CurrentDID())
		return nil
	},
}
