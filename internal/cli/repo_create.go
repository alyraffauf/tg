package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/alyraffauf/tg/knot"
	"github.com/spf13/cobra"
)

func newRepoCreateCommand(service *app.Service) *cobra.Command {
	var description, knotHost, pushPath, remote string
	var clone bool

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a repository on Tangled",
		Long: `Create a repository on Tangled.

The repository is provisioned on a knot (default ` + knot.DefaultKnot + `) and a
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
				selectedKnot = knot.DefaultKnot
			}

			uri, handle, err := service.ProvisionRepo(ctx, app.ProvisionRepoInput{
				KnotHost:    selectedKnot,
				Name:        args[0],
				Description: description,
			})
			if err != nil {
				return err
			}

			result := app.RepoCreateResult{
				Handle: handle,
				Name:   args[0],
				URI:    uri,
				Knot:   selectedKnot,
			}

			if clone {
				if _, err := service.CloneRepo(ctx, app.CloneRepoInput{
					Handle:      handle,
					Repo:        args[0],
					Destination: args[0],
				}); err != nil {
					return fmt.Errorf("clone new repository: %w", err)
				}
				result.Cloned = true
			}
			if pushPath != "" {
				if err := pushToNewRepo(ctx, service, cmd.ErrOrStderr(), pushToNewRepoInput{
					KnotHost:   selectedKnot,
					RepoURI:    uri,
					Handle:     handle,
					RepoName:   args[0],
					PushPath:   pushPath,
					RemoteName: remote,
				}); err != nil {
					return err
				}
				result.Pushed = true
			}

			return output(cmd, result, func(repo app.RepoCreateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Created repository %s/%s\n", repo.Handle, repo.Name)
				if repo.Cloned {
					fmt.Fprintf(cmd.OutOrStdout(), "Cloned into %s\n", repo.Name)
				}
				if repo.Pushed {
					fmt.Fprintf(cmd.OutOrStdout(), "Pushed to %s\n", repo.Name)
				}
			})
		},
	}
	command.Flags().StringVar(&description, "description", "", "Repository description")
	command.Flags().StringVar(&knotHost, "knot", "", "knot host to create on (default "+knot.DefaultKnot+")")
	command.Flags().BoolVar(&clone, "clone", false, "Clone the new repository into the current directory")
	command.Flags().StringVar(&pushPath, "push", "", "Push an existing local repository at this path to the new remote (e.g. .)")
	command.Flags().StringVar(&remote, "remote", "origin", "Remote name to use with --push")
	return command
}

type pushToNewRepoInput struct {
	KnotHost   string
	RepoURI    string
	Handle     string
	RepoName   string
	PushPath   string
	RemoteName string
}

// pushToNewRepo sets the default branch then pushes. Default-branch failure is
// warned, not fatal. Set before push so the knot's hook skips its PR suggestion.
func pushToNewRepo(ctx context.Context, service *app.Service, errorWriter io.Writer, in pushToNewRepoInput) error {
	branch, defaultBranchErr, err := service.PushNewRepo(ctx, app.PushNewRepoInput{
		KnotHost:   in.KnotHost,
		RepoURI:    in.RepoURI,
		Dir:        in.PushPath,
		Handle:     in.Handle,
		Repo:       in.RepoName,
		RemoteName: in.RemoteName,
	})
	if defaultBranchErr != nil {
		fmt.Fprintf(errorWriter, "warning: could not set default branch: %v\n", defaultBranchErr)
	} else {
		fmt.Fprintf(errorWriter, "Set default branch to %s\n", branch)
	}
	return err
}
