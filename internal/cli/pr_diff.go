package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRDiffCommand(service *app.Service) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   "diff <rkey>",
		Short: "Print the latest patch for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			patch, err := service.PullPatch(ctx, target, args[0])
			if err != nil {
				return err
			}
			if _, err := cmd.OutOrStdout().Write(patch.Patch); err != nil {
				return fmt.Errorf("write patch: %w", err)
			}
			return nil
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
