package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRViewCommand(service *app.Service) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   "view <record-key>",
		Short: "View a pull request",
		Long: `View a pull request by its record key, the last part of its at:// URI.

Without --repo, tg uses the origin remote in the current directory.`,
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
				fields := []detailField{
					{"Title", view.Title},
					{"Status", formatDetailState(cmd.OutOrStdout(), view.State)},
					{"Author", view.Author.Handle},
					{"Created", localTimestamp(view.CreatedAt)},
					{"Branch", view.SourceBranch + " → " + view.TargetBranch},
				}
				renderDetail(cmd.OutOrStdout(), fields, view.Body)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
