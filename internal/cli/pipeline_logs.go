package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

type pipelineLogsService interface {
	TargetFromCWD(context.Context) (app.Target, error)
	PipelineStatus(context.Context, app.Target) (*app.PipelineStatusResult, error)
	PipelineLogs(context.Context, app.Target, string, []string, func(app.PipelineLogEvent) error) error
}

var workflowColors = []color.Color{
	lipgloss.Cyan, lipgloss.Magenta, lipgloss.Green,
	lipgloss.Yellow, lipgloss.Blue, lipgloss.Red,
}

func newPipelineLogsCommand(service pipelineLogsService) *cobra.Command {
	var repository string
	var workflows []string

	command := &cobra.Command{
		Use:   "logs [id]",
		Short: "Stream pipeline logs",
		Long: `Stream log output from a pipeline in real time.

If no pipeline ID is given, streams the latest pipeline for the repository. If
--repo is not set, the repository is detected from the current directory's git
origin remote. Use --workflow to filter to specific workflows.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			pipelineID := ""
			if len(args) > 0 {
				pipelineID = args[0]
			} else {
				status, err := service.PipelineStatus(cmd.Context(), target)
				if err != nil {
					return err
				}
				pipelineID = status.Pipeline.ID
			}
			jsonOutput, _ := cmd.Flags().GetBool("json")
			out := cmd.OutOrStdout()
			terminal := isTerminal(out)
			return service.PipelineLogs(cmd.Context(), target, pipelineID, workflows, func(event app.PipelineLogEvent) error {
				if jsonOutput {
					return json.NewEncoder(out).Encode(event)
				}
				return renderPipelineLogEvent(out, cmd.ErrOrStderr(), event, terminal)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringSliceVarP(&workflows, "workflow", "w", nil, "Workflow name to stream (repeatable)")
	return command
}

func renderPipelineLogEvent(stdout, stderr io.Writer, event app.PipelineLogEvent, terminal bool) error {
	switch {
	case event.Control != nil:
		if event.Control.Status == "start" {
			renderStepHeader(stdout, event.Control, terminal)
		}
	case event.Data != nil:
		w := stderr
		if event.Data.Stream == "stdout" {
			w = stdout
		}
		return writeLogLines(w, event.Data, terminal)
	}
	return nil
}

func renderStepHeader(w io.Writer, control *app.PipelineLogControl, terminal bool) {
	sep := "──"
	tag := control.Workflow
	if terminal {
		sep = lipgloss.NewStyle().Faint(true).Render(sep)
		tag = lipgloss.NewStyle().Foreground(workflowColor(control.Workflow)).Render(tag)
	}
	fmt.Fprintf(w, "\n%s %s: %s ", sep, tag, control.Content)
	if control.Command != nil {
		command := "$ " + *control.Command
		if terminal {
			command = lipgloss.NewStyle().Faint(true).Render(command)
		}
		fmt.Fprintf(w, "(%s) ", command)
	}
	fmt.Fprintf(w, "%s\n", sep)
}

func writeLogLines(w io.Writer, data *app.PipelineLogData, terminal bool) error {
	content := strings.TrimRight(data.Content, "\n")
	if content == "" {
		return nil
	}
	tag := data.Workflow
	if terminal {
		tag = lipgloss.NewStyle().Foreground(workflowColor(data.Workflow)).Render(tag)
	}
	for _, line := range strings.Split(content, "\n") {
		if terminal && data.Stream == "stderr" {
			line = lipgloss.NewStyle().Foreground(lipgloss.Yellow).Render(line)
		}
		fmt.Fprintf(w, "[%s] %s\n", tag, line)
	}
	return nil
}

func workflowColor(workflow string) color.Color {
	var hash int
	for _, r := range workflow {
		hash += int(r)
	}
	return workflowColors[hash%len(workflowColors)]
}
