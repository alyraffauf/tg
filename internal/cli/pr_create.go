package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRCreateCommand(service *app.Service) *cobra.Command {
	var title, bodyText, bodyFile, base, head, repository, sourceRepository string

	command := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request from the current branch",
		Long: "Create a pull request by uploading a gzipped git patch and writing a sh.tangled.repo.pull record. " +
			"By default, the current repository and branch are both the source and target repository, and origin's " +
			"default branch is the target branch. Use --repo and --source-repo for a fork-based pull request.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			body, err := commandBody(bodyText, bodyFile)
			if err != nil {
				return err
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
				return err
			}
			return output(cmd, result, func(created *app.PRCreateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Created pull request %s (%s -> %s)\n", created.URI, created.Head, created.Base)
			})
		},
	}
	command.Flags().StringVarP(&title, "title", "t", "", "Pull request title")
	command.Flags().StringVarP(&bodyText, "body", "b", "", "Pull request body")
	command.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read pull request body from file")
	command.Flags().StringVarP(&base, "base", "B", "", "Target branch (default: origin's default branch)")
	command.Flags().StringVarP(&head, "head", "H", "", "Source branch (default: current branch)")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringVar(&sourceRepository, "source-repo", "", "Source repository as handle/repo (for fork-based pull requests)")
	_ = command.MarkFlagRequired("title")
	return command
}
