package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

func newRepoCloneCommand(service *app.Service, defaultProtocol string) *cobra.Command {
	return &cobra.Command{
		Use:   "clone <repo | handle/repo | repo-did> [directory]",
		Short: "Clone a Tangled repository",
		Long: `Clone a Tangled repository into a local directory.

The default destination is the repository name. If only a repository name is
given, the authenticated user's handle is used. If a record supplies a name
that is unsafe as a default directory, provide the directory argument.

Run "tg auth login" first when using the repository-only form. A repository DID
does not require a tg login. The clone's origin remote uses the DID.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			input := app.CloneRepoInput{Protocol: defaultProtocol}
			if _, err := syntax.ParseDID(args[0]); err == nil {
				input.RepoDID = args[0]
			} else {
				target, err := resolveCloneTarget(ctx, args[0], service)
				if err != nil {
					return err
				}
				input.Handle = target.Handle
				input.Repo = target.Repo
			}
			if len(args) == 2 {
				input.Destination = args[1]
			}

			result, err := service.CloneRepo(ctx, input)
			if err != nil {
				return fmt.Errorf("clone %q: %w", args[0], err)
			}
			return output(cmd, result, func(clone *app.RepoCloneResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Cloned %s/%s into %s\n", clone.Handle, clone.Repo, clone.Destination)
				for _, warning := range clone.Warnings {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
				}
			})
		},
	}
}
