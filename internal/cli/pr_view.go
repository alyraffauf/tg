package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRViewCommand(service *app.Service) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   "view <rkey>",
		Short: "View a pull request for a Tangled repository",
		Long: `View a pull request by its rkey (the last segment of its at:// URI).

If --repo is not set, the repository is detected from the current
directory's git origin remote.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			view, err := service.ViewPull(ctx, target, args[0])
			if err != nil {
				return err
			}
			return output(cmd, view, func(view *app.ViewResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Title:   %s\n", view.Title)
				fmt.Fprintf(cmd.OutOrStdout(), "Author:  %s\n", view.Author.Handle)
				fmt.Fprintf(cmd.OutOrStdout(), "Created: %s\n", view.CreatedAt)
				fmt.Fprintf(cmd.OutOrStdout(), "Branch:  %s → %s\n", view.SourceBranch, view.TargetBranch)
				if view.Body != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", view.Body)
				}
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
