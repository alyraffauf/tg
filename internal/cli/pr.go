package cli

import "github.com/spf13/cobra"

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests on a Tangled repository",
}
