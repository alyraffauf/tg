package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestRenderPipelineList(t *testing.T) {
	var output bytes.Buffer
	renderPipelineList(&output, []app.Pipeline{{
		ID:        "pipeline-123",
		Commit:    "0123456789abcdef",
		CreatedAt: "2026-07-30T12:00:00Z",
		Trigger: map[string]any{
			"$type": "org.tangled.event.push",
			"ref":   "refs/heads/main",
		},
		Workflows: []app.PipelineWorkflow{
			{Name: "test", Status: "success"},
			{Name: "lint", Status: "success"},
		},
	}})

	for _, expected := range []string{"TRIGGER", "push refs/heads/main", "✓ 2 passed", "0123456", "2026-07-30"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output missing %q:\n%s", expected, output.String())
		}
	}
}

func TestPipelineStatusSummary(t *testing.T) {
	originalIsTerminal := isTerminal
	t.Cleanup(func() { isTerminal = originalIsTerminal })

	workflows := []app.PipelineWorkflow{
		{Name: "test", Status: "success"},
		{Name: "lint", Status: "running"},
		{Name: "build", Status: "failed"},
	}
	got := pipelineStatusSummary(&bytes.Buffer{}, workflows)
	const want = "✗ 1 failed · ● 1 running · ✓ 1 passed"
	if got != want {
		t.Fatalf("pipelineStatusSummary() = %q, want %q", got, want)
	}

	isTerminal = func(io.Writer) bool { return true }
	colored := pipelineStatusSummary(&bytes.Buffer{}, workflows)
	if !strings.Contains(colored, "\x1b[") {
		t.Fatalf("terminal summary has no color: %q", colored)
	}
}

func TestPipelineTrigger(t *testing.T) {
	trigger := map[string]any{
		"$type":        "org.tangled.event.pullRequest",
		"sourceBranch": "feature",
		"targetBranch": "main",
	}
	if got := pipelineTrigger(trigger); got != "pull request feature → main" {
		t.Fatalf("pipelineTrigger() = %q", got)
	}
}

func TestPipelineScheduleAndApprovalDisplay(t *testing.T) {
	if got := pipelineTrigger(map[string]any{"$type": "org.tangled.ci.trigger.schedule", "ref": "refs/heads/main"}); got != "schedule refs/heads/main" {
		t.Fatalf("trigger=%s", got)
	}
	if got := pipelineStatusSummary(&bytes.Buffer{}, []app.PipelineWorkflow{{Status: "blocked"}}); got != "○ 1 awaiting approval" {
		t.Fatalf("status=%s", got)
	}
	if got := workflowStatusLabel("blocked"); got != "○ awaiting approval" {
		t.Fatalf("label=%s", got)
	}
}
