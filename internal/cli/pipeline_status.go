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
		Short: "Show the default branch's latest pipeline",
		Long: `Show the pipeline for the latest commit on a repository's default branch.

Without a repository, tg uses the origin remote in the current directory.`,
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
