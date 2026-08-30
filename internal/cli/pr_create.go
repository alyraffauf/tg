package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

type pullCreateService interface {
	CreatePull(context.Context, app.CreatePullInput) (*app.PRCreateResult, error)
	TargetFromCWD(context.Context) (app.Target, error)
}

func newPRCreateCommand(service pullCreateService) *cobra.Command {
	var title, bodyText, bodyFile, base, head, repository, sourceRepository string

	command := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request from the current branch",
		Long: `Create a pull request by uploading a gzipped git patch and writing a
sh.tangled.repo.pull record. By default, the current repository and branch are
both the source and target repository, and origin's default branch is the
target branch. Use --repo and --source-repo for a fork-based pull request.

When title and body are omitted, tg opens $EDITOR. The first line is the title,
the second line must be blank, and the remaining text is the body.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			body, err := commandBody(bodyText, bodyFile)
			if err != nil {
				return err
			}
			openEditor := !cmd.Flags().Changed("title") && !cmd.Flags().Changed("body") && !cmd.Flags().Changed("body-file")
			if !cmd.Flags().Changed("title") && !openEditor {
				return fmt.Errorf("provide --title when --body or --body-file is used")
			}
			repoDir, err := getwd()
			if err != nil {
				return err
			}
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			var source *app.Target
			if sourceRepository != "" {
				st, err := app.ParseTarget(sourceRepository)
				if err != nil {
					return err
				}
				source = &st
			}
			draft := titledDraft{}
			if openEditor {
				draft, err = editTitledDraft(ctx, "pull request", "tg-pull-request-*.md", titledDraftTemplate, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
				if errors.Is(err, errTitledDraftCanceled) {
					fmt.Fprintln(cmd.ErrOrStderr(), "Pull request creation canceled.")
					return nil
				}
				if err != nil {
					return err
				}
				title = draft.Title
				body = draft.Body
			}
			result, err := service.CreatePull(ctx, app.CreatePullInput{
				RepoDir: repoDir,
				Title:   title,
				Body:    body,
				Base:    base,
				Head:    head,
				Target:  target,
				Source:  source,
			})
			if err != nil {
				if draft.Path != "" {
					return fmt.Errorf("%w; draft saved to %s", err, draft.Path)
				}
				return err
			}
			if draft.Path != "" {
				removeTitledDraft(draft.Path, "pull request", cmd.ErrOrStderr())
			}
			return output(cmd, result, func(created *app.PRCreateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Created pull request %s (%s -> %s)\n", created.URI, created.Head, created.Base)
			})
		},
	}
	command.Flags().StringVarP(&title, "title", "t", "", "Pull request title")
	command.Flags().StringVarP(&bodyText, "body", "b", "", "Pull request body")
	command.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read pull request body from file")
	command.Flags().StringVarP(&base, "base", "B", "", "Target branch or local remote-tracking branch (required for forks; default: origin's default branch)")
	command.Flags().StringVarP(&head, "head", "H", "", "Source branch (default: current branch)")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringVar(&sourceRepository, "source-repo", "", "Source repository as handle/repo (for fork-based pull requests)")
	return command
}
