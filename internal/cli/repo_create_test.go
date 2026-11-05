package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	_ = output(command, &app.RepoCreateResult{
		Handle: "owner.test", Name: "example", Pushed: true, DefaultBranch: "main",
		Warnings: []string{"could not set another setting"},
	}, func(repo *app.RepoCreateResult) { renderRepoCreate(command, repo) })

	if got := stdout.String(); !strings.Contains(got, "Pushed to example\n") {
		t.Fatalf("stdout = %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "Set default branch to main\n") || !strings.Contains(got, "warning: could not set another setting\n") {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRepoCreateRendersReceiptBeforeReturningLocalFailure(t *testing.T) {
	for _, jsonMode := range []bool{false, true} {
		t.Run(fmt.Sprint(jsonMode), func(t *testing.T) {
			failure := errors.New("clone failed")
			receipt := &app.RepoCreateResult{Handle: "owner.test", Name: "example", URI: "at://did:plc:repo/org.tangled.repo.manifest/self", RepoDID: "did:plc:repo", Declared: true, Warnings: []string{"indexing pending"}}
			command := newRepoCreateCommand(func(context.Context, app.CreateRepoInput) (*app.RepoCreateResult, error) { return receipt, failure }, "", "22", "https")
			var stdout, stderr bytes.Buffer
			command.SetOut(&stdout)
			command.SetErr(&stderr)
			command.Flags().Bool("json", jsonMode, "")
			command.SetArgs([]string{"example", "--clone"})
			command.SilenceErrors, command.SilenceUsage = true, true
			if err := command.Execute(); !errors.Is(err, failure) {
				t.Fatalf("err=%v", err)
			}
			if jsonMode {
				var result app.RepoCreateResult
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.URI != receipt.URI || !result.Declared || result.Cloned || len(result.Warnings) != 1 {
					t.Fatalf("result=%+v", result)
				}
			} else if !strings.Contains(stdout.String(), "owner.test/example") || !strings.Contains(stderr.String(), "indexing pending") {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}
