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
	"github.com/spf13/cobra"
)

func TestParseCommentDraft(t *testing.T) {
	tests := []struct {
		name     string
		document string
		want     string
	}{
		{name: "body", document: "A comment\n\nWith detail\n", want: "A comment\n\nWith detail"},
		{name: "indented code", document: "    code\n", want: "    code"},
		{name: "trailing spaces preserved", document: "A comment  \n", want: "A comment  "},
		{name: "CRLF", document: "A comment\r\n\r\nWith detail\r\n", want: "A comment\n\nWith detail"},
		{name: "instructions removed", document: "A comment\n" + commentDraftSentinel + "\nignored", want: "A comment"},
		{name: "other HTML comment retained", document: "<!-- keep this -->", want: "<!-- keep this -->"},
		{name: "empty", document: commentDraftTemplate},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parseCommentDraft(test.document); got != test.want {
				t.Fatalf("parseCommentDraft() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCommentCommandsOpenEditorWhenBodyIsOmitted(t *testing.T) {
	tests := []struct {
		name       string
		newCommand func(*testCommentService) *cobra.Command
		wantKind   string
	}{
		{name: "issue", newCommand: func(service *testCommentService) *cobra.Command { return newIssueCommentCommand(service) }, wantKind: "issue"},
		{name: "pull request", newCommand: func(service *testCommentService) *cobra.Command { return newPRCommentCommand(service) }, wantKind: "pull"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pathLog := filepath.Join(t.TempDir(), "path")
			t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "A comment\n", 0))
			service := &testCommentService{}
			command := test.newCommand(service)
			command.SetArgs([]string{"123", "--repo", "alice.example/project"})
			command.SetIn(nil)
			command.SetOut(io.Discard)
			command.SetErr(io.Discard)

			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if service.kind != test.wantKind || service.body != "A comment" {
				t.Fatalf("submission = kind %q, body %q", service.kind, service.body)
			}
			pathBytes, err := os.ReadFile(pathLog)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(string(pathBytes)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("submitted draft still exists: %v", err)
			}
		})
	}
}

func TestCommentCommandExplicitBodyDoesNotOpenEditor(t *testing.T) {
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
	service := &testCommentService{}
	command := newIssueCommentCommand(service)
	command.SetArgs([]string{"123", "--body", "Explicit comment", "--repo", "alice.example/project"})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.body != "Explicit comment" {
		t.Fatalf("body = %q", service.body)
	}
}

func TestCommentCommandRetainsDraftWhenSubmissionFails(t *testing.T) {
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "Unsubmitted comment\n", 0))
	service := &testCommentService{commentError: errors.New("network unavailable")}
	command := newIssueCommentCommand(service)
	command.SetArgs([]string{"123", "--repo", "alice.example/project"})
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

type testCommentService struct {
	commentError error
	kind         string
	body         string
}

func (service *testCommentService) CommentIssue(_ context.Context, _ app.Target, _, body string) (*app.CreatedRecordResult, error) {
	service.kind = "issue"
	service.body = body
	return service.commentResult()
}

func (service *testCommentService) CommentPull(_ context.Context, _ app.Target, _, body string) (*app.CreatedRecordResult, error) {
	service.kind = "pull"
	service.body = body
	return service.commentResult()
}

func (service *testCommentService) commentResult() (*app.CreatedRecordResult, error) {
	if service.commentError != nil {
		return nil, service.commentError
	}
	return &app.CreatedRecordResult{URI: "at://did:plc:owner/sh.tangled.feed.comment/123"}, nil
}

func (*testCommentService) TargetFromCWD(context.Context) (app.Target, error) {
	return app.Target{Handle: "alice.example", Repo: "project"}, nil
}
