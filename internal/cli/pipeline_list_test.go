package cli

import (
	"bytes"
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
			"$type": "sh.tangled.ci.trigger#push",
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
	workflows := []app.PipelineWorkflow{
		{Name: "test", Status: "success"},
		{Name: "lint", Status: "running"},
		{Name: "build", Status: "failed"},
	}
	got := pipelineStatusSummary(workflows)
	const want = "✗ 1 failed · ● 1 running · ✓ 1 passed"
	if got != want {
		t.Fatalf("pipelineStatusSummary() = %q, want %q", got, want)
	}
}

func TestPipelineTrigger(t *testing.T) {
	trigger := map[string]any{
		"$type":        "sh.tangled.ci.trigger#pullRequest",
		"sourceBranch": "feature",
		"targetBranch": "main",
	}
	if got := pipelineTrigger(trigger); got != "pull request feature → main" {
		t.Fatalf("pipelineTrigger() = %q", got)
	}
}
