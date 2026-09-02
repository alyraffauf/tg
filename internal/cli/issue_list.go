package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

type listFlags struct {
	author string
	state  string
	limit  int64
	order  string
}

func (flags *listFlags) add(command *cobra.Command) {
	command.Flags().StringVar(&flags.author, "author", "", "Filter by author handle or DID")
	command.Flags().StringVar(&flags.state, "state", "", "Filter by state")
	command.Flags().Int64Var(&flags.limit, "limit", 0, "Maximum number of results")
	command.Flags().StringVar(&flags.order, "order", "", "Sort order: asc or desc")
}

func (flags listFlags) options(command *cobra.Command) (app.ListOptions, error) {
	if command.Flags().Changed("limit") && flags.limit <= 0 {
		return app.ListOptions{}, fmt.Errorf("list limit must be positive")
	}
	return app.ListOptions{
		Author: flags.author,
		State:  flags.state,
		Limit:  flags.limit,
		Order:  flags.order,
	}, nil
}

func newIssueListCommand(service *app.Service) *cobra.Command {
	var flags listFlags
	command := &cobra.Command{
		Use:   "list [handle/repo]",
		Short: "List issues for a Tangled repository",
		Long: `List issues for a Tangled repository.

If no argument is given, the command detects the repository from the
"origin" remote URL of the git repository in the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := flags.options(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			target, err := resolveTarget(ctx, args, service)
			if err != nil {
				return err
			}
			result, err := service.ListIssues(ctx, target, options)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.ItemListResult) {
				renderList(cmd.OutOrStdout(), result.Items, "No issues found.")
				renderRecordWarnings(cmd.ErrOrStderr(), result.Warnings)
			})
		},
	}
	flags.add(command)
	return command
}
