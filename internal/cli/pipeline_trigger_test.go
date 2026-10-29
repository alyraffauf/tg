package cli

import (
	"bytes"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestPipelineTriggerCommandFlags(t *testing.T) {
	command := newPipelineTriggerCommand(&app.Service{})
	if command.Use != "trigger <revision>" {
		t.Fatalf("command use = %q", command.Use)
	}
	for _, name := range []string{"repo", "workflow"} {
		if command.Flags().Lookup(name) == nil {
			t.Errorf("pipeline trigger has no --%s flag", name)
		}
	}
}

func TestPipelineTriggerReportsEveryPipelineAndEmptyResults(t *testing.T) {
	for _, test := range []struct {
		name   string
		result app.PipelineTriggerResult
		want   string
	}{
		{name: "multiple", result: app.PipelineTriggerResult{Pipeline: "first", Pipelines: []string{"first", "second"}, Message: "Started verification"}, want: "Triggered pipeline first.\nTriggered pipeline second.\nStarted verification\n"},
		{name: "none", result: app.PipelineTriggerResult{Message: "No matching workflows"}, want: "No pipelines were scheduled.\nNo matching workflows\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			renderPipelineTrigger(&output, &test.result)
			if output.String() != test.want {
				t.Fatalf("output = %q, want %q", output.String(), test.want)
			}
		})
	}
}
