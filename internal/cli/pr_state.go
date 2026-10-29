package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRUpdateCommand(service *app.Service) *cobra.Command {
	var base, head, repository, sourceRepository string

	command := &cobra.Command{
		Use:   "update <record-key>",
		Short: "Submit a new round for a pull request",
		Long:  "Submit a new immutable patch round from published source commits. Push the source branch first; tg does not push it automatically.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoDir, err := getwd()
			if err != nil {
				return err
			}
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			var source *app.Target
			if sourceRepository != "" {
				parsed, err := app.ParseTarget(sourceRepository)
				if err != nil {
					return err
				}
				source = &parsed
			}
			result, err := service.UpdatePullRound(cmd.Context(), app.UpdatePullRoundInput{
				Target: target, Source: source, Head: head,
				RepoDir: repoDir,
				Rkey:    args[0],
				Base:    base,
			})
			if err != nil {
				return err
			}
			return output(cmd, result, func(*app.UpdatedRecordResult) {})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringVar(&sourceRepository, "source-repo", "", "Source repository as handle/repo")
	command.Flags().StringVarP(&head, "head", "H", "", "Published source branch")
	command.Flags().StringVarP(&base, "base", "B", "", "Target branch or local remote-tracking branch. Required for forks.")
	return command
}

func newPRCloseCommand(service *app.Service) *cobra.Command {
	return newPRStateCommand(service, "close", "closed")
}

func newPRReopenCommand(service *app.Service) *cobra.Command {
	return newPRStateCommand(service, "reopen", "open")
}

func newPREditCommand(service *app.Service) *cobra.Command {
	var titleText, bodyText, repository, expectedCID string

	command := &cobra.Command{
		Use:   "edit <record-key>",
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
				return fmt.Errorf("provide --title or --body")
			}
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			result, err := service.EditPull(cmd.Context(), target, args[0], title, body, expectedCID)
			if err != nil {
				return err
			}
			return output(cmd, result, func(*app.UpdatedRecordResult) {})
		},
	}
	command.Flags().StringVar(&expectedCID, "expected-cid", "", "Reject changes unless this record CID is current")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringVarP(&titleText, "title", "t", "", "New title")
	command.Flags().StringVarP(&bodyText, "body", "b", "", "New body")
	return command
}

func newPRMergeCommand(service *app.Service) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   "merge <record-key>",
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
			return output(cmd, result, func(result *app.MergePullResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Pull request %s merged\n", result.Rkey)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}

func newPRStateCommand(service *app.Service, use, status string) *cobra.Command {
	var repository, expectedCID string

	command := &cobra.Command{
		Use:   use + " <record-key>",
		Short: use + " a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.SetPullState(ctx, target, args[0], status, expectedCID)
			if err != nil {
				return fmt.Errorf("%s pull request: %w", use, err)
			}
			return output(cmd, result, func(result *app.StateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Pull request %s %s\n", result.Rkey, result.State)
			})
		},
	}
	command.Flags().StringVar(&expectedCID, "expected-cid", "", "Reject changes unless this record CID is current")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
