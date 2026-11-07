package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

type testRepoForkService struct {
	result *app.RepoForkResult
	err    error
}

func (service testRepoForkService) ForkRepoOnKnot(context.Context, app.Target, string, string) (*app.RepoForkResult, error) {
	return service.result, service.err
}

func TestRepoForkPrintsRetainedRepositoryBeforeReturningFailure(t *testing.T) {
	failure := errors.New("declaration write lost")
	receipt := &app.RepoForkResult{RepoDID: "did:plc:fork", Handle: "owner.test", Name: "fork", URI: "at://did:plc:fork/org.tangled.repo.manifest/self", Knot: "knot.example"}
	command := newRepoForkCommand(testRepoForkService{result: receipt, err: failure})
	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&bytes.Buffer{})
	command.Flags().Bool("json", true, "")
	command.SetArgs([]string{"alice.test/project", "fork"})
	command.SilenceErrors, command.SilenceUsage = true, true

	if err := command.Execute(); !errors.Is(err, failure) {
		t.Fatalf("err = %v, want %v", err, failure)
	}
	var result app.RepoForkResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout is not a fork result: %v: %q", err, stdout.String())
	}
	if result.RepoDID != "did:plc:fork" || result.Declared {
		t.Fatalf("result = %+v, want undeclared did:plc:fork", result)
	}
}

func TestPullCreateReportsRetainedTicketBeforeReturningFailure(t *testing.T) {
	failure := errors.New("submit patch: response lost")
	receipt := &app.PRCreateResult{PatchStatus: app.PatchUnknown, URI: "at://did:plc:repo/sh.tangled.repo.issue/123", CID: "bafy", Title: "Pull title", Base: "main", Head: "feature"}

	t.Run("json", func(t *testing.T) {
		command := newPRCreateCommand(&testPullCreateService{createError: failure, partial: receipt})
		var stdout bytes.Buffer
		command.SetOut(&stdout)
		command.SetErr(&bytes.Buffer{})
		command.Flags().Bool("json", true, "")
		command.SetArgs([]string{"--title", "Pull title", "--repo", "alice.example/project"})
		command.SilenceErrors, command.SilenceUsage = true, true

		if err := command.Execute(); !errors.Is(err, failure) {
			t.Fatalf("err = %v, want %v", err, failure)
		}
		var result app.PRCreateResult
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("stdout is not a pull result: %v: %q", err, stdout.String())
		}
		if result.URI != receipt.URI || result.PatchStatus != app.PatchUnknown {
			t.Fatalf("result = %+v, want unknown patch status for %s", result, receipt.URI)
		}
	})

	t.Run("human keeps draft and does not claim success", func(t *testing.T) {
		pathLog := filepath.Join(t.TempDir(), "path")
		t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "Pull title\n\nPull details\n", 0))
		command := newPRCreateCommand(&testPullCreateService{createError: failure, partial: receipt})
		var stdout, stderr bytes.Buffer
		command.SetOut(&stdout)
		command.SetErr(&stderr)
		command.SetIn(nil)
		command.SetArgs([]string{"--repo", "alice.example/project"})
		command.SilenceErrors, command.SilenceUsage = true, true

		err := command.Execute()
		if !errors.Is(err, failure) {
			t.Fatalf("err = %v, want %v", err, failure)
		}
		if strings.Contains(stdout.String(), "Created pull request") {
			t.Fatalf("stdout claims success: %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), receipt.URI) {
			t.Fatalf("stderr does not name the retained ticket: %q", stderr.String())
		}
		draftPath, readErr := os.ReadFile(pathLog)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, statErr := os.Stat(string(draftPath)); statErr != nil {
			t.Fatalf("draft was removed after a failed submission: %v", statErr)
		}
	})
}
