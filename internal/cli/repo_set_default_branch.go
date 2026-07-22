package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoSetDefaultBranchCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "set-default-branch <branch> [handle/repo]",
		Short: "Set a Tangled repository's default branch",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			branch := args[0]
			target, err := resolveTarget(ctx, args[1:], service)
			if err != nil {
				return err
			}
			result, err := service.SetRepoDefaultBranch(ctx, target, branch)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.RepoDefaultBranchResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Set default branch for %s to %s\n", result.URI, result.Branch)
			})
		},
	}
}
