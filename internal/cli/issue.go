package cli

import "github.com/spf13/cobra"

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues on a Tangled repository",
}
