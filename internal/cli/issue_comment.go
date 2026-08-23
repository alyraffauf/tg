package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

type issueCommentService interface {
	CommentIssue(context.Context, app.Target, string, string) (*app.CreatedRecordResult, error)
	TargetFromCWD(context.Context) (app.Target, error)
}

func newIssueCommentCommand(service issueCommentService) *cobra.Command {
	var bodyText, bodyFile, repository string

	command := &cobra.Command{
		Use:   "comment <rkey>",
		Short: "Add a comment to an issue",
		Long:  "Add a comment to an issue. When the body is omitted, tg opens $EDITOR.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			body, err := commandBody(bodyText, bodyFile)
			if err != nil {
				return err
			}
			draft := commentDraft{}
			openEditor := !cmd.Flags().Changed("body") && !cmd.Flags().Changed("body-file")
			if !openEditor && body == "" {
				return fmt.Errorf("provide --body or --body-file")
			}
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			if openEditor {
				draft, err = editCommentDraft(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
				if errors.Is(err, errCommentCreationCanceled) {
					fmt.Fprintln(cmd.ErrOrStderr(), "Comment creation canceled.")
					return nil
				}
				if err != nil {
					return err
				}
				body = draft.Body
			}
			result, err := service.CommentIssue(ctx, target, args[0], body)
			if err != nil {
				return commentSubmissionError(err, draft)
			}
			if draft.Path != "" {
				removeCommentDraft(draft.Path, cmd.ErrOrStderr())
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
