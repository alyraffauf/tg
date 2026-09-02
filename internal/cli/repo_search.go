package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoSearchCommand(service *app.Service) *cobra.Command {
	var limit int64
	command := &cobra.Command{
		Use:   "search <query>",
		Short: "Search repositories on Tangled",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.SearchRepos(cmd.Context(), args[0], limit)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.RepoListResult) {
				renderTable(cmd.OutOrStdout(), []string{"NAME", "KNOT", "DESCRIPTION", "STARS"}, repoSearchRows(result.Items), "No repositories found.")
				renderRecordWarnings(cmd.ErrOrStderr(), result.Warnings)
			})
		},
	}
	command.Flags().Int64VarP(&limit, "limit", "n", 50, "Maximum number of results")
	return command
}

func repoSearchRows(items []app.RepoItem) [][]string {
	rows := repoRows(items, true)
	for i, repo := range items {
		rows[i][0] = repo.Author + "/" + repo.Name
	}
	return rows
}
