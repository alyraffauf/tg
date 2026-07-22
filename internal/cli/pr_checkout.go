package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRCheckoutCommand(service *app.Service) *cobra.Command {
	var repository, branch string
	var force bool

	command := &cobra.Command{
		Use:   "checkout <rkey>",
		Short: "Check out a pull request in Git",
		Long:  "Check out the latest pull request round on the current remote target branch.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			rkey := args[0]
			repoDir, err := getwd()
			if err != nil {
				return err
			}
			var target *app.Target
			if repository != "" {
				parsedTarget, err := app.ParseTarget(repository)
				if err != nil {
					return err
				}
				target = &parsedTarget
			}
			result, err := service.CheckoutPull(ctx, app.CheckoutPullInput{
				RepoDir: repoDir,
				Rkey:    rkey,
				Target:  target,
				Branch:  branch,
				Force:   force,
			})
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.PRCheckoutResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Checked out pull request %s as branch %s\n", result.Rkey, result.Branch)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringVarP(&branch, "branch", "b", "", "Local branch name (default: pr-<rkey>)")
	command.Flags().BoolVarP(&force, "force", "f", false, "Reset an existing checkout branch")
	return command
}
