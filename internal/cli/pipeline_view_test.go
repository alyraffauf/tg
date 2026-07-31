package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestRenderPipelineDetail(t *testing.T) {
	pipeline := &app.Pipeline{
		ID:        "pipeline-123",
		Commit:    "0123456789abcdef",
		CreatedAt: "2026-07-30T12:00:00Z",
		Trigger:   map[string]any{"$type": "sh.tangled.ci.trigger#manual", "ref": "main"},
		Workflows: []app.PipelineWorkflow{{
			Name: "test", Status: "failed", StartedAt: "2026-07-30T12:01:00Z",
			FinishedAt: "2026-07-30T12:02:00Z", Error: "exit status 1",
		}},
	}

	var output bytes.Buffer
	renderPipelineDetail(&output, pipeline)
	for _, expected := range []string{"ID:", "pipeline-123", "Trigger:", "manual main", "WORKFLOW", "✗ failed", "exit status 1"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output missing %q:\n%s", expected, output.String())
		}
	}
	if strings.Contains(output.String(), "Workflows") {
		t.Errorf("output contains redundant workflow heading:\n%s", output.String())
	}
}

func TestPipelineViewRepoFlag(t *testing.T) {
	command := newPipelineViewCommand(&app.Service{})
	flag := command.Flags().Lookup("repo")
	if flag == nil {
		t.Fatal("pipeline view has no --repo flag")
	}
	if flag.Shorthand != "R" || flag.Usage != "Target repository as handle/repo" {
		t.Fatalf("repo flag = %+v", flag)
	}
	if command.Use != "view <id>" {
		t.Fatalf("command use = %q, want view <id>", command.Use)
	}
}
