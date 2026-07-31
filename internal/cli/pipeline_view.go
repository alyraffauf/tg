package cli

import (
	"fmt"
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPipelineViewCommand(service *app.Service) *cobra.Command {
	var repository string

	command := &cobra.Command{
		Use:   "view <id>",
		Short: "View a pipeline for a Tangled repository",
		Long: `View a pipeline by its spindle-local ID.

If --repo is not set, the repository is detected from the current directory's
git origin remote.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			pipeline, err := service.ViewPipeline(cmd.Context(), target, args[0])
			if err != nil {
				return err
			}
			return output(cmd, pipeline, func(pipeline *app.Pipeline) {
				renderPipelineDetail(cmd.OutOrStdout(), pipeline)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}

func renderPipelineDetail(writer io.Writer, pipeline *app.Pipeline) {
	fields := []detailField{
		{"ID", pipeline.ID},
		{"Commit", pipeline.Commit},
		{"Trigger", pipelineTrigger(pipeline.Trigger)},
		{"Created", localTimestamp(pipeline.CreatedAt)},
	}
	if pipeline.SourceRepo != "" {
		fields = append(fields, detailField{"Source Repo", pipeline.SourceRepo})
	}
	renderDetail(writer, fields, "")

	renderPipelineWorkflows(writer, pipeline.Workflows)
}

func renderPipelineWorkflows(writer io.Writer, workflows []app.PipelineWorkflow) {
	rows := make([][]string, 0, len(workflows))
	for _, workflow := range workflows {
		rows = append(rows, []string{
			workflow.Name,
			workflowStatusLabel(workflow.Status),
			localTimestamp(workflow.StartedAt),
			localTimestamp(workflow.FinishedAt),
			workflow.Error,
		})
	}
	fmt.Fprintln(writer)
	renderTable(writer, []string{"WORKFLOW", "STATUS", "STARTED", "FINISHED", "ERROR"}, rows, "No workflows found.")
}

func workflowStatusLabel(status string) string {
	switch status {
	case "success":
		return "✓ success"
	case "failed", "timeout":
		return "✗ " + status
	case "running":
		return "● running"
	case "pending":
		return "○ pending"
	case "cancelled":
		return "⊘ cancelled"
	default:
		return status
	}
}
