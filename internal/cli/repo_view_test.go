package cli

import (
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestRepoViewHasNoPipelineFlag(t *testing.T) {
	command := newRepoViewCommand(&app.Service{})
	flag := command.Flags().Lookup("pipeline")
	if flag != nil {
		t.Fatal("repo view should include pipeline status automatically")
	}
}
