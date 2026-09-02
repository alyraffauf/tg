package cli

import (
	"fmt"
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoListCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list [handle]",
		Short: "List repositories owned by a Tangled user",
		Long: `List repositories owned by a Tangled user.

If no argument is given, lists the authenticated user's repositories
(run "tg auth login" first).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			handle, err := resolveHandleOrSelf(ctx, args, service)
			if err != nil {
				return err
			}
			result, err := service.ListRepos(ctx, handle)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.RepoListResult) {
				renderRepoList(cmd.OutOrStdout(), result.Items)
				renderRecordWarnings(cmd.ErrOrStderr(), result.Warnings)
			})
		},
	}
}

func renderRepoList(writer io.Writer, items []app.RepoItem) {
	rows := repoRows(items, false)
	renderTable(writer, []string{"NAME", "KNOT", "DESCRIPTION", "CREATED"}, rows, "No repositories found.")
}

func repoRows(items []app.RepoItem, includeStars bool) [][]string {
	rows := make([][]string, 0, len(items))
	for _, repo := range items {
		row := []string{repo.Name, repo.Knot, repo.Description}
		if includeStars {
			row = append(row, fmt.Sprint(repo.Stars))
		} else {
			row = append(row, shortDate(repo.CreatedAt))
		}
		rows = append(rows, row)
	}
	return rows
}
