package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newIssueViewCommand(service *app.Service) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   "view <rkey>",
		Short: "View an issue for a Tangled repository",
		Long: `View an issue by its rkey (the last segment of its at:// URI).

If --repo is not set, the repository is detected from the current
directory's git origin remote.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			view, err := service.ViewIssue(ctx, target, args[0])
			if err != nil {
				return err
			}
			return output(cmd, view, func(view *app.ViewResult) {
				fields := []detailField{
					{"Title", view.Title},
					{"Author", view.Author.Handle},
					{"Created", view.CreatedAt},
				}
				renderDetail(cmd.OutOrStdout(), fields, view.Body)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
