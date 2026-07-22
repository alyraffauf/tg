package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newIssueCloseCommand(service *app.Service) *cobra.Command {
	return newIssueStateCommand(service, "close", "closed")
}

func newIssueReopenCommand(service *app.Service) *cobra.Command {
	return newIssueStateCommand(service, "reopen", "open")
}

func newIssueEditCommand(service *app.Service) *cobra.Command {
	var titleText, bodyText string

	command := &cobra.Command{
		Use:   "edit <rkey>",
		Short: "Edit an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var title, body *string
			if cmd.Flags().Changed("title") {
				title = &titleText
			}
			if cmd.Flags().Changed("body") {
				body = &bodyText
			}
			if title == nil && body == nil {
				return fmt.Errorf("set --title or --body")
			}
			return service.EditIssue(cmd.Context(), args[0], title, body)
		},
	}
	command.Flags().StringVarP(&titleText, "title", "t", "", "New title")
	command.Flags().StringVarP(&bodyText, "body", "b", "", "New body")
	return command
}

func newIssueStateCommand(service *app.Service, use, state string) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   use + " <rkey>",
		Short: use + " an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.SetIssueState(ctx, target, args[0], state)
			if err != nil {
				return fmt.Errorf("%s issue: %w", use, err)
			}
			return output(cmd, result, func(result *app.StateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Issue %s %s\n", result.Rkey, result.State)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
