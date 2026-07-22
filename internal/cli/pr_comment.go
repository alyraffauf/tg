package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRCommentCommand(service *app.Service) *cobra.Command {
	var bodyText, bodyFile, repository string

	command := &cobra.Command{
		Use:   "comment <rkey>",
		Short: "Add a comment to a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := commandBody(bodyText, bodyFile)
			if err != nil {
				return err
			}
			if body == "" {
				return fmt.Errorf("set --body or --body-file")
			}
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.CommentPull(ctx, target, args[0], body)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.CreatedRecordResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Added comment %s\n", result.URI)
			})
		},
	}
	command.Flags().StringVarP(&bodyText, "body", "b", "", "Comment body")
	command.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read comment body from file")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
