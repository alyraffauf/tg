package cli

import (
	"io"
	"strconv"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPipelineListCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list [handle/repo]",
		Short: "List pipelines for a Tangled repository",
		Long: `List pipelines for a Tangled repository.

If no argument is given, the command detects the repository from the
"origin" remote URL of the git repository in the current directory.`,
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
			pipelineStatusSummary(pipeline.Workflows),
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
	switch triggerType {
	case "sh.tangled.ci.trigger#push":
		return joinTriggerDetails("push", triggerString(trigger, "ref"))
	case "sh.tangled.ci.trigger#pullRequest":
		sourceBranch := triggerString(trigger, "sourceBranch")
		targetBranch := triggerString(trigger, "targetBranch")
		if sourceBranch != "" && targetBranch != "" {
			return "pull request " + sourceBranch + " → " + targetBranch
		}
		return "pull request"
	case "sh.tangled.ci.trigger#manual":
		return joinTriggerDetails("manual", triggerString(trigger, "ref"))
	default:
		return "unknown"
	}
}

func triggerString(trigger map[string]any, key string) string {
	value, _ := trigger[key].(string)
	return value
}

func joinTriggerDetails(kind, detail string) string {
	if detail == "" {
		return kind
	}
	return kind + " " + detail
}

func pipelineStatusSummary(workflows []app.PipelineWorkflow) string {
	counts := make(map[string]int)
	for _, workflow := range workflows {
		counts[workflow.Status]++
	}
	return strings.Join(statusSummaryParts(counts), " · ")
}

func statusSummaryParts(counts map[string]int) []string {
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
			parts = append(parts, status.symbol+" "+pluralizeStatus(count, status.label))
		}
	}
	if len(parts) == 0 {
		return []string{"unknown"}
	}
	return parts
}

func pluralizeStatus(count int, label string) string {
	if count == 1 {
		return "1 " + label
	}
	return strconv.Itoa(count) + " " + label
}
