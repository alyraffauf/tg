package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoCreateCommand(service *app.Service) *cobra.Command {
	var description, knotHost, pushPath, remote string
	var clone bool

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a repository on Tangled",
		Long: `Create a repository on Tangled.

The repository is provisioned on a knot (default ` + app.DefaultKnot + `) and a
sh.tangled.repo record is written to your PDS. The repository name is used as
the record key, matching the current Tangled schema.

Use --clone to clone the new repository into the current directory, or
--push=<path> to push an existing local repository at that path to the new
remote (and set its current branch as the default branch).

Requires authentication (run "tg auth login" first).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			selectedKnot := knotHost
			if selectedKnot == "" {
				selectedKnot = app.DefaultKnot
			}

			result, err := service.CreateRepo(ctx, app.CreateRepoInput{
				KnotHost: selectedKnot, Name: args[0], Description: description,
				Clone: clone, PushPath: pushPath, RemoteName: remote,
			})
			if err != nil {
				return err
			}
			return output(cmd, result, func(repo *app.RepoCreateResult) { renderRepoCreate(cmd, repo) })
		},
	}
	command.Flags().StringVar(&description, "description", "", "Repository description")
	command.Flags().StringVar(&knotHost, "knot", "", "knot host to create on (default "+app.DefaultKnot+")")
	command.Flags().BoolVar(&clone, "clone", false, "Clone the new repository into the current directory")
	command.Flags().StringVar(&pushPath, "push", "", "Push an existing local repository at this path to the new remote (e.g. .)")
	command.Flags().StringVar(&remote, "remote", "origin", "Remote name to use with --push")
	return command
}

func renderRepoCreate(cmd *cobra.Command, repo *app.RepoCreateResult) {
	fmt.Fprintf(cmd.OutOrStdout(), "Created repository %s/%s\n", repo.Handle, repo.Name)
	if repo.Cloned {
		fmt.Fprintf(cmd.OutOrStdout(), "Cloned into %s\n", repo.Name)
	}
	if repo.Pushed {
		fmt.Fprintf(cmd.OutOrStdout(), "Pushed to %s\n", repo.Name)
	}
	if repo.DefaultBranch != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Set default branch to %s\n", repo.DefaultBranch)
	}
	for _, warning := range repo.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
	}
}
