package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

type issueCreateService interface {
	CreateIssue(context.Context, app.Target, string, string) (*app.CreatedRecordResult, error)
	TargetFromCWD(context.Context) (app.Target, error)
}

func newIssueCreateCommand(service issueCreateService) *cobra.Command {
	var bodyText, bodyFile, repository string

	command := &cobra.Command{
		Use:   "create [title]",
		Short: "Create an issue on a Tangled repository",
		Long: `Create an issue on a Tangled repository. When title and body are omitted,
tg opens $EDITOR. The edited document's first line is the title, its second line
must be blank, and the remaining text is the required body.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if len(args) == 0 {
				if cmd.Flags().Changed("body") || cmd.Flags().Changed("body-file") {
					return fmt.Errorf("title is required when --body or --body-file is used")
				}
				target, err := resolveTargetFlag(ctx, repository, service)
				if err != nil {
					return err
				}
				draft, err := editIssueDraft(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
				if errors.Is(err, errIssueCreationCanceled) {
					fmt.Fprintln(cmd.ErrOrStderr(), "Issue creation canceled.")
					return nil
				}
				if err != nil {
					return err
				}
				result, err := service.CreateIssue(ctx, target, draft.Title, draft.Body)
				if err != nil {
					return fmt.Errorf("%w; draft saved to %s", err, draft.Path)
				}
				removeIssueDraft(draft.Path, cmd.ErrOrStderr())
				return output(cmd, result, func(result *app.CreatedRecordResult) {
					fmt.Fprintf(cmd.OutOrStdout(), "Created issue %s\n", result.URI)
				})
			}

			body, err := commandBody(bodyText, bodyFile)
			if err != nil {
				return err
			}
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.CreateIssue(ctx, target, args[0], body)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.CreatedRecordResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Created issue %s\n", result.URI)
			})
		},
	}
	command.Flags().StringVarP(&bodyText, "body", "b", "", "Issue body")
	command.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read issue body from file")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
