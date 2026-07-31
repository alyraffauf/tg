package cli

import (
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
