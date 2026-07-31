package cli

import (
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestPipelineStatusCommand(t *testing.T) {
	command := newPipelineStatusCommand(&app.Service{})
	if command.Use != "status [handle/repo]" {
		t.Fatalf("command use = %q", command.Use)
	}
	if command.Args == nil {
		t.Fatal("pipeline status has no argument validator")
	}
}
