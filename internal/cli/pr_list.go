package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRListCommand(service *app.Service) *cobra.Command {
	var flags listFlags
	command := &cobra.Command{
		Use:   "list [handle/repo]",
		Short: "List pull requests for a Tangled repository",
		Long: `List pull requests for a Tangled repository.

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
			result, err := service.ListPulls(ctx, target, options)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.ItemListResult) {
				renderList(cmd.OutOrStdout(), result.Items, "No pull requests found.")
				renderRecordWarnings(cmd.ErrOrStderr(), result.Warnings)
			})
		},
	}
	flags.add(command)
	return command
}
