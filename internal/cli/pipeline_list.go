package cli

import (
	"io"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPipelineListCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list [handle/repo]",
		Short: "List pipelines",
		Long: `List pipelines for a repository.

Without a repository, tg uses the origin remote in the current directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTarget(cmd.Context(), args, service)
			if err != nil {
				return err
			}
			pipelines, err := service.ListPipelines(cmd.Context(), target)
			if err != nil {
				return err
			}
			return output(cmd, pipelines, func(pipelines []app.Pipeline) {
				renderPipelineList(cmd.OutOrStdout(), pipelines)
			})
		},
	}
}

func renderPipelineList(writer io.Writer, pipelines []app.Pipeline) {
	rows := make([][]string, 0, len(pipelines))
	for _, pipeline := range pipelines {
		rows = append(rows, []string{
			pipeline.ID,
			shortCommit(pipeline.Commit),
			pipelineTrigger(pipeline.Trigger),
			pipelineStatusSummary(writer, pipeline.Workflows),
			shortDate(pipeline.CreatedAt),
		})
	}
	renderTable(writer, []string{"ID", "COMMIT", "TRIGGER", "STATUS", "CREATED"}, rows, "No pipelines found.")
}

func shortCommit(commit string) string {
	if len(commit) <= 7 {
		return commit
	}
	return commit[:7]
}

func pipelineTrigger(trigger map[string]any) string {
	triggerType, _ := trigger["$type"].(string)
	ref, _ := trigger["ref"].(string)
	switch triggerType {
	case "sh.tangled.ci.trigger#push":
		if ref == "" {
			return "push"
		}
		return "push " + ref
	case "sh.tangled.ci.trigger#pullRequest":
		sourceBranch, _ := trigger["sourceBranch"].(string)
		targetBranch, _ := trigger["targetBranch"].(string)
		if sourceBranch != "" && targetBranch != "" {
			return "pull request " + sourceBranch + " → " + targetBranch
		}
		return "pull request"
	case "sh.tangled.ci.trigger#manual":
		if ref == "" {
			return "manual"
		}
		return "manual " + ref
	default:
		return "unknown"
	}
}

func pipelineStatusSummary(writer io.Writer, workflows []app.PipelineWorkflow) string {
	counts := make(map[string]int)
	for _, workflow := range workflows {
		counts[workflow.Status]++
	}
	statuses := []struct {
		name   string
		symbol string
		label  string
	}{
		{"failed", "✗", "failed"},
		{"timeout", "✗", "timed out"},
		{"running", "●", "running"},
		{"pending", "○", "pending"},
		{"cancelled", "⊘", "cancelled"},
		{"success", "✓", "passed"},
	}

	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if count := counts[status.name]; count > 0 {
			part := status.symbol + " " + strconv.Itoa(count) + " " + status.label
			if terminalColor := workflowStatusColor(status.name); isTerminal(writer) && terminalColor != nil {
				part = lipgloss.NewStyle().Foreground(terminalColor).Render(part)
			}
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, " · ")
}
