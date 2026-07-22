package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoDeleteCommand(service *app.Service) *cobra.Command {
	var confirm bool

	command := &cobra.Command{
		Use:   "delete [handle/repo]",
		Short: "Delete a Tangled repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("refusing to delete without --yes")
			}
			ctx := cmd.Context()
			target, err := resolveTarget(ctx, args, service)
			if err != nil {
				return err
			}
			result, err := service.DeleteRepo(ctx, target)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.RepoDeleteResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Deleted repository %s\n", result.URI)
			})
		},
	}
	command.Flags().BoolVar(&confirm, "yes", false, "Confirm permanent repository deletion")
	return command
}
