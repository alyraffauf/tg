package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoCloneCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "clone <repo | handle/repo> [directory]",
		Short: "Clone a Tangled repository",
		Long: `Clone a Tangled repository via SSH into a local directory.

The default destination is the repository name. If only a repository name is
given, the authenticated user's handle is used.

Run "tg auth login" first when using the repository-only form.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveCloneTarget(ctx, args[0], service)
			if err != nil {
				return err
			}

			dest := target.Repo
			if len(args) == 2 {
				dest = args[1]
			}

			result, err := service.CloneRepo(ctx, app.CloneRepoInput{
				Handle:      target.Handle,
				Repo:        target.Repo,
				Destination: dest,
			})
			if err != nil {
				return fmt.Errorf("clone %q: %w", args[0], err)
			}
			return output(cmd, result, func(clone *app.RepoCloneResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Cloned %s/%s into %s\n", clone.Handle, clone.Repo, clone.Destination)
			})
		},
	}
}
