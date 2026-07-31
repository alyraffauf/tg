package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPipelineCancelCommand(service *app.Service) *cobra.Command {
	var repository string
	var workflows []string

	command := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Cancel a pipeline or selected workflows",
		Long: `Cancel every workflow in a pipeline, or only the workflows selected with --workflow.

If --repo is not set, the repository is detected from the current directory's
git origin remote.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			result, err := service.CancelPipeline(cmd.Context(), target, args[0], workflows)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.PipelineCancelResult) {
				renderPipelineCancellation(cmd.OutOrStdout(), result, len(workflows) > 0)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringSliceVarP(&workflows, "workflow", "w", nil, "Workflow name to cancel (repeatable)")
	return command
}

func renderPipelineCancellation(writer io.Writer, result *app.PipelineCancelResult, selectedWorkflows bool) {
	if !result.CancellationRequested {
		if selectedWorkflows {
			fmt.Fprintln(writer, "None of the selected workflows are pending or running.")
			return
		}
		fmt.Fprintf(writer, "Pipeline %s has no pending or running workflows.\n", result.Pipeline)
		return
	}
	if !selectedWorkflows {
		fmt.Fprintf(writer, "Cancellation requested for pipeline %s.\n", result.Pipeline)
		return
	}
	fmt.Fprintf(writer, "Cancellation requested for workflows %s in pipeline %s.\n", strings.Join(result.Workflows, ", "), result.Pipeline)
}
