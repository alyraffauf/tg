package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRListCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list [handle/repo]",
		Short: "List pull requests for a Tangled repository",
		Long: `List pull requests for a Tangled repository.

If no argument is given, the command detects the repository from the
"origin" remote URL of the git repository in the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTarget(ctx, args, service)
			if err != nil {
				return err
			}
			items, err := service.ListPulls(ctx, target)
			if err != nil {
				return err
			}
			return output(cmd, items, func(items []app.Item) {
				renderList(cmd.OutOrStdout(), items, "No pull requests found.")
			})
		},
	}
}
