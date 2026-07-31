package cli

import (
	"bytes"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestPipelineCancelCommandFlags(t *testing.T) {
	command := newPipelineCancelCommand(&app.Service{})
	if command.Use != "cancel <id>" {
		t.Fatalf("command use = %q", command.Use)
	}
	for _, name := range []string{"repo", "workflow"} {
		if command.Flags().Lookup(name) == nil {
			t.Errorf("pipeline cancel has no --%s flag", name)
		}
	}
}

func TestRenderPipelineCancellationForUncancellableSelection(t *testing.T) {
	var output bytes.Buffer
	renderPipelineCancellation(&output, &app.PipelineCancelResult{Pipeline: "pipeline-123"}, true)

	const want = "None of the selected workflows are pending or running.\n"
	if output.String() != want {
		t.Fatalf("renderPipelineCancellation() = %q, want %q", output.String(), want)
	}
}
