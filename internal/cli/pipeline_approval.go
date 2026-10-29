package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPipelineApproveCommand(service *app.Service) *cobra.Command {
	var repository string
	command := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve blocked pipeline workflows",
		Long:  "Allow workflows waiting for approval to run. Review the pipeline's source before approving it. Without --repo, tg uses the origin remote.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			result, err := service.ApprovePipeline(cmd.Context(), target, args[0])
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.PipelineApproveResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Approved pipeline %s.\n", result.Pipeline)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}

func newPipelineRetryCommand(service *app.Service) *cobra.Command {
	var repository string
	var workflows []string
	command := &cobra.Command{
		Use:   "retry <id>",
		Short: "Rerun a pipeline's stored workflows",
		Long:  "Create a new run from a pipeline's stored specification. Use --workflow to select workflows. Blocked pipelines need approval first. Without --repo, tg uses the origin remote.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			result, err := service.RetryPipeline(cmd.Context(), target, args[0], workflows)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.PipelineRetryResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Created pipeline %s from %s.\n", result.Pipeline, result.OriginalPipeline)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringSliceVarP(&workflows, "workflow", "w", nil, "Workflow to rerun (repeatable)")
	return command
}
