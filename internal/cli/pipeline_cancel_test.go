package cli

import (
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
