package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRCloseCommand(service *app.Service) *cobra.Command {
	return newPRStateCommand(service, "close", "closed")
}

func newPRReopenCommand(service *app.Service) *cobra.Command {
	return newPRStateCommand(service, "reopen", "open")
}

func newPREditCommand(service *app.Service) *cobra.Command {
	var titleText, bodyText string

	command := &cobra.Command{
		Use:   "edit <rkey>",
		Short: "Edit a pull request",
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
			return service.EditPull(cmd.Context(), args[0], title, body)
		},
	}
	command.Flags().StringVarP(&titleText, "title", "t", "", "New title")
	command.Flags().StringVarP(&bodyText, "body", "b", "", "New body")
	return command
}

func newPRMergeCommand(service *app.Service) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   "merge <rkey>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.MergePull(ctx, target, args[0])
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.StateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Pull request %s merged\n", result.Rkey)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}

func newPRStateCommand(service *app.Service, use, status string) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   use + " <rkey>",
		Short: use + " a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.SetPullState(ctx, target, args[0], status)
			if err != nil {
				return fmt.Errorf("%s pull request: %w", use, err)
			}
			return output(cmd, result, func(result *app.StateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Pull request %s %s\n", result.Rkey, result.State)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
