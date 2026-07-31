package cli

import (
	"errors"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

var errPipelineFailed = errors.New("pipeline failed")

func newPipelineStatusCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "status [handle/repo]",
		Short: "Show the latest pipeline status for a repository",
		Long: `Show the latest pipeline status for a Tangled repository.

If no repository argument is given, the command detects the repository from the
"origin" remote URL of the git repository in the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTarget(cmd.Context(), args, service)
			if err != nil {
				return err
			}
			status, err := service.PipelineStatus(cmd.Context(), target)
			if err != nil {
				return err
			}
			if err := output(cmd, status.Pipeline, func(pipeline *app.Pipeline) {
				renderPipelineDetail(cmd.OutOrStdout(), pipeline)
			}); err != nil {
				return err
			}
			if status.HasFailures {
				return errPipelineFailed
			}
			return nil
		},
	}
}
