package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestPullCreateOpensEditorWhenTitleAndBodyAreOmitted(t *testing.T) {
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "Pull title\n\nPull details\n", 0))
	service := &testPullCreateService{}
	command := newPRCreateCommand(service)
	command.SetArgs([]string{"--repo", "alice.example/project"})
	command.SetIn(nil)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.input.Title != "Pull title" || service.input.Body != "Pull details" {
		t.Fatalf("CreatePull() input = %+v", service.input)
	}
	pathBytes, err := os.ReadFile(pathLog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(pathBytes)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("submitted draft still exists: %v", err)
	}
}

func TestPullCreateAllowsEmptyEditorBody(t *testing.T) {
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "Pull title\n\n", 0))
	service := &testPullCreateService{}
	command := newPRCreateCommand(service)
	command.SetArgs([]string{"--repo", "alice.example/project"})
	command.SetIn(nil)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.input.Title != "Pull title" || service.input.Body != "" {
		t.Fatalf("CreatePull() input = %+v", service.input)
	}
}

func TestPullCreateExplicitTitleDoesNotOpenEditor(t *testing.T) {
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
	service := &testPullCreateService{}
	command := newPRCreateCommand(service)
	command.SetArgs([]string{"--title", "Pull title", "--body", "Pull details", "--repo", "alice.example/project"})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.input.Title != "Pull title" || service.input.Body != "Pull details" {
		t.Fatalf("CreatePull() input = %+v", service.input)
	}
}

func TestPullCreateBodyWithoutTitleDoesNotOpenEditor(t *testing.T) {
	command := newPRCreateCommand(nil)
	command.SetArgs([]string{"--body", "Pull details"})
	command.SetErr(io.Discard)
	err := command.Execute()
	if err == nil || err.Error() != "provide --title when --body or --body-file is used" {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestPullCreateRetainsDraftWhenSubmissionFails(t *testing.T) {
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "Pull title\n\nUnsubmitted details\n", 0))
	service := &testPullCreateService{createError: errors.New("network unavailable")}
	command := newPRCreateCommand(service)
	command.SetArgs([]string{"--repo", "alice.example/project"})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	err := command.Execute()
	if err == nil || !strings.Contains(err.Error(), "network unavailable; draft saved to") {
		t.Fatalf("Execute() error = %v", err)
	}
	pathBytes, readError := os.ReadFile(pathLog)
	if readError != nil {
		t.Fatal(readError)
	}
	path := string(pathBytes)
	t.Cleanup(func() { _ = os.Remove(path) })
	if _, statError := os.Stat(path); statError != nil {
		t.Fatalf("saved draft %q: %v", path, statError)
	}
}

type testPullCreateService struct {
	createError error
	input       app.CreatePullInput
}

func (service *testPullCreateService) CreatePull(_ context.Context, input app.CreatePullInput) (*app.PRCreateResult, error) {
	service.input = input
	if service.createError != nil {
		return nil, service.createError
	}
	return &app.PRCreateResult{URI: "at://did:plc:owner/sh.tangled.repo.pull/123", Base: "main", Head: "feature"}, nil
}

func (*testPullCreateService) TargetFromCWD(context.Context) (app.Target, error) {
	return app.Target{Handle: "alice.example", Repo: "project"}, nil
}
