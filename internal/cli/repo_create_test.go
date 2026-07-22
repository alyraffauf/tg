package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func TestRenderRepoCreateReportsDefaultBranchAndWarnings(t *testing.T) {
	command := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)

	renderRepoCreate(command, &app.RepoCreateResult{
		Handle: "owner.test", Name: "example", Pushed: true, DefaultBranch: "main",
		Warnings: []string{"could not set another setting"},
	})

	if got := stdout.String(); !strings.Contains(got, "Pushed to example\n") {
		t.Fatalf("stdout = %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "Set default branch to main\n") || !strings.Contains(got, "warning: could not set another setting\n") {
		t.Fatalf("stderr = %q", got)
	}
}
