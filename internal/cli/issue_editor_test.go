package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestParseIssueDraft(t *testing.T) {
	tests := []struct {
		name      string
		document  string
		wantTitle string
		wantBody  string
		wantError string
	}{
		{name: "title only", document: "Bug report\n", wantError: "body must not be empty"},
		{name: "empty body", document: "Bug report\n\n", wantError: "body must not be empty"},
		{name: "body", document: "Bug report\n\nSteps to reproduce\n\nMore detail\n", wantTitle: "Bug report", wantBody: "Steps to reproduce\n\nMore detail"},
		{name: "indented code body", document: "Bug report\n\n    code\n", wantTitle: "Bug report", wantBody: "    code"},
		{name: "trailing spaces preserved", document: "Bug report\n\nDetails  \n", wantTitle: "Bug report", wantBody: "Details  "},
		{name: "CRLF", document: "Bug report\r\n\r\nDetails\r\n", wantTitle: "Bug report", wantBody: "Details"},
		{name: "instructions removed", document: "Bug report\n\nDetails\n" + issueDraftSentinel + "\nignored", wantTitle: "Bug report", wantBody: "Details"},
		{name: "other HTML comment retained", document: "Bug report\n\n<!-- keep this -->", wantTitle: "Bug report", wantBody: "<!-- keep this -->"},
		{name: "untouched template cancels", document: issueDraftTemplate, wantError: errIssueCreationCanceled.Error()},
		{name: "whitespace cancels", document: " \n\n\t", wantError: errIssueCreationCanceled.Error()},
		{name: "missing second line", document: "Bug report", wantError: "title must be followed by an empty line"},
		{name: "nonempty second line", document: "Bug report\nDetails", wantError: "second line must be empty"},
		{name: "empty title with body", document: "\n\nDetails", wantError: "title must not be empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			title, body, err := parseIssueDraft(test.document)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("parseIssueDraft() error = %v, want containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseIssueDraft() error = %v", err)
			}
			if title != test.wantTitle || body != test.wantBody {
				t.Fatalf("parseIssueDraft() = (%q, %q), want (%q, %q)", title, body, test.wantTitle, test.wantBody)
			}
		})
	}
}

func TestIssueCreateBodyWithoutTitleDoesNotOpenEditor(t *testing.T) {
	command := newIssueCreateCommand(nil)
	command.SetArgs([]string{"--body", "details"})
	command.SetErr(io.Discard)
	err := command.Execute()
	if err == nil || err.Error() != "title is required when --body or --body-file is used" {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestEditIssueDraftLifecycle(t *testing.T) {
	tests := []struct {
		name       string
		document   string
		exitStatus int
		wantTitle  string
		wantBody   string
		wantError  string
		wantSaved  bool
	}{
		{name: "valid draft", document: "Bug report\n\nDetails\n", wantTitle: "Bug report", wantBody: "Details", wantSaved: true},
		{name: "canceled draft", document: issueDraftTemplate, wantError: errIssueCreationCanceled.Error()},
		{name: "empty body", document: "Bug report\n\n", wantError: "body must not be empty", wantSaved: true},
		{name: "malformed draft", document: "Bug report\nDetails\n", wantError: "second line must be empty", wantSaved: true},
		{name: "failed editor", document: "Unfinished\n\nDraft\n", exitStatus: 23, wantError: "run issue editor", wantSaved: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pathLog := filepath.Join(t.TempDir(), "path")
			editorPath := writeDraftEditor(t, pathLog, test.document, test.exitStatus)
			t.Setenv("EDITOR", editorPath)

			draft, err := editIssueDraft(context.Background(), nil, io.Discard, io.Discard)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("editIssueDraft() error = %v, want containing %q", err, test.wantError)
				}
			} else if err != nil {
				t.Fatalf("editIssueDraft() error = %v", err)
			}
			if draft.Title != test.wantTitle || draft.Body != test.wantBody {
				t.Fatalf("editIssueDraft() = %+v, want title %q and body %q", draft, test.wantTitle, test.wantBody)
			}

			pathBytes, err := os.ReadFile(pathLog)
			if err != nil {
				t.Fatal(err)
			}
			path := string(pathBytes)
			_, statError := os.Stat(path)
			if test.wantSaved && statError != nil {
				t.Fatalf("saved draft %q: %v", path, statError)
			}
			if !test.wantSaved && !errors.Is(statError, os.ErrNotExist) {
				t.Fatalf("canceled draft %q still exists", path)
			}
			if test.wantSaved {
				t.Cleanup(func() { _ = os.Remove(path) })
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if permissions := info.Mode().Perm(); permissions != 0o600 {
					t.Fatalf("draft permissions = %o, want 600", permissions)
				}
			}
		})
	}
}

func TestIssueCreateRetainsEditedDraftWhenSubmissionFails(t *testing.T) {
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "Bug report\n\nDetails\n", 0))
	service := &testIssueCreateService{createError: errors.New("network unavailable")}
	command := newIssueCreateCommand(service)
	command.SetArgs([]string{"--repo", "alice.example/project"})
	command.SetIn(nil)
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
	if service.title != "Bug report" || service.body != "Details" {
		t.Fatalf("submission = (%q, %q)", service.title, service.body)
	}
}

func TestIssueCreateRemovesEditedDraftAfterSubmission(t *testing.T) {
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "Bug report\n\nDetails\n", 0))
	service := &testIssueCreateService{}
	command := newIssueCreateCommand(service)
	command.SetArgs([]string{"--repo", "alice.example/project"})
	command.SetIn(nil)
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	pathBytes, err := os.ReadFile(pathLog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(pathBytes)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("submitted draft still exists: %v", err)
	}
}

func TestIssueCreateExplicitInvocationDoesNotOpenEditor(t *testing.T) {
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
	service := &testIssueCreateService{}
	command := newIssueCreateCommand(service)
	command.SetArgs([]string{"Bug report", "--body", "Details", "--repo", "alice.example/project"})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.title != "Bug report" || service.body != "Details" {
		t.Fatalf("submission = (%q, %q)", service.title, service.body)
	}
}

func TestIssueCreateExplicitTitleOnlyRemainsValid(t *testing.T) {
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
	service := &testIssueCreateService{}
	command := newIssueCreateCommand(service)
	command.SetArgs([]string{"Bug report", "--repo", "alice.example/project"})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.title != "Bug report" || service.body != "" {
		t.Fatalf("submission = (%q, %q)", service.title, service.body)
	}
}

func TestRemoveIssueDraftWarnsWhenRemovalFails(t *testing.T) {
	var errorOutput strings.Builder
	path := t.TempDir()
	if err := os.WriteFile(filepath.Join(path, "draft"), []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	removeIssueDraft(path, &errorOutput)
	if !strings.Contains(errorOutput.String(), "warning: remove issue draft") {
		t.Fatalf("warning = %q", errorOutput.String())
	}
}

type testIssueCreateService struct {
	createError error
	title       string
	body        string
}

func (service *testIssueCreateService) CreateIssue(_ context.Context, _ app.Target, title, body string) (*app.CreatedRecordResult, error) {
	service.title = title
	service.body = body
	if service.createError != nil {
		return nil, service.createError
	}
	return &app.CreatedRecordResult{URI: "at://did:plc:owner/sh.tangled.repo.issue/123"}, nil
}

func (*testIssueCreateService) TargetFromCWD(context.Context) (app.Target, error) {
	return app.Target{Handle: "alice.example", Repo: "project"}, nil
}

func writeDraftEditor(t *testing.T, pathLog, document string, exitStatus int) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "editor.sh")
	contents := "#!/bin/sh\n" +
		"printf '%s' \"$1\" > " + shellQuote(pathLog) + "\n" +
		"cat > \"$1\" <<'TG_ISSUE_DRAFT'\n" + document + "TG_ISSUE_DRAFT\n" +
		"exit " + fmt.Sprint(exitStatus) + "\n"
	if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	return script
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
