package cli

import "github.com/spf13/cobra"

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories on Tangled",
}
