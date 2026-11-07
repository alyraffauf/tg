package cli

import (
	"context"
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

type issueEditService interface {
	cwdTargetResolver
	EditIssue(context.Context, app.EditTicketInput) (*app.UpdatedRecordResult, error)
}

func newIssueEditCommand(service issueEditService) *cobra.Command {
	var titleText, bodyText, repository, expectedCID string
	var labels app.LabelEditInput

	command := &cobra.Command{
		Use:   "edit <record-key>",
		Short: "Edit an issue",
		Long:  "Edit an issue's title, body, or labels. Label names must match exactly one repository definition.\n\nLabel edits also drop references to deleted definitions and report a warning.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var title, body *string
			if cmd.Flags().Changed("title") {
				title = &titleText
			}
			if cmd.Flags().Changed("body") {
				body = &bodyText
			}
			if title == nil && body == nil && len(labels.AddLabels) == 0 && len(labels.RemoveLabels) == 0 {
				return fmt.Errorf("provide --title, --body, --add-label or --remove-label")
			}
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			result, err := service.EditIssue(cmd.Context(), app.EditTicketInput{Target: target, Rkey: args[0], Title: title, Body: body, Labels: labels, ExpectedCID: expectedCID})
			if err != nil {
				return err
			}
			return output(cmd, result, func(*app.UpdatedRecordResult) {})
		},
	}
	command.Flags().StringArrayVar(&labels.AddLabels, "add-label", nil, "Exact label name to add (repeatable)")
	command.Flags().StringArrayVar(&labels.RemoveLabels, "remove-label", nil, "Exact label name to remove (repeatable)")
	command.Flags().StringVar(&expectedCID, "expected-cid", "", "Reject changes unless this record CID is current")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringVarP(&titleText, "title", "t", "", "New title")
	command.Flags().StringVarP(&bodyText, "body", "b", "", "New body")
	return command
}

func newIssueStateCommand(service *app.Service, use, state string) *cobra.Command {
	var repository, expectedCID string

	command := &cobra.Command{
		Use:   use + " <record-key>",
		Short: use + " an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.SetIssueState(ctx, target, args[0], state, expectedCID)
			if err != nil {
				return fmt.Errorf("%s issue: %w", use, err)
			}
			return output(cmd, result, func(result *app.StateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Issue %s %s\n", result.Rkey, result.State)
			})
		},
	}
	command.Flags().StringVar(&expectedCID, "expected-cid", "", "Reject changes unless this record CID is current")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
