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

func TestStringContents(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.md")
	if err := os.WriteFile(file, []byte("# hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "binary.bin")
	if err := os.WriteFile(binary, []byte{0xff, 0xfe, 0xfd}, 0o600); err != nil {
		t.Fatal(err)
	}
	oversize := filepath.Join(dir, "big.txt")
	f, err := os.Create(oversize)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxStringContents + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		stdin        string
		args         []string
		filenameFlag string
		wantContents string
		wantFilename string
		wantErr      bool
	}{
		{name: "file with basename", args: []string{file}, wantContents: "# hello", wantFilename: "hello.md"},
		{name: "file with flag override", args: []string{file}, filenameFlag: "custom.md", wantContents: "# hello", wantFilename: "custom.md"},
		{name: "missing file", args: []string{filepath.Join(dir, "nope")}, wantErr: true},
		{name: "empty file", args: []string{empty}, wantErr: true},
		{name: "non-UTF-8 file", args: []string{binary}, wantErr: true},
		{name: "oversize file", args: []string{oversize}, wantErr: true},
		{name: "stdin without filename", stdin: "# hello", wantErr: true},
		{name: "stdin with filename", stdin: "# from stdin", args: []string{"-"}, filenameFlag: "stdin.md", wantContents: "# from stdin", wantFilename: "stdin.md"},
		{name: "stdin with filename but empty", args: []string{"-"}, filenameFlag: "stdin.md", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents, filename, err := stringContents(strings.NewReader(tt.stdin), tt.args, tt.filenameFlag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("stringContents() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if contents != tt.wantContents {
				t.Errorf("contents = %q, want %q", contents, tt.wantContents)
			}
			if filename != tt.wantFilename {
				t.Errorf("filename = %q, want %q", filename, tt.wantFilename)
			}
		})
	}
}

func TestStringCreateOpensEditorForTerminalInput(t *testing.T) {
	originalIsTerminalInput := isTerminalInput
	t.Cleanup(func() { isTerminalInput = originalIsTerminalInput })
	isTerminalInput = func(io.Reader) bool { return true }
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "package main\n", 0))
	service := &testStringCreateService{}
	command := newStringCreateCommand(service)
	command.SetArgs([]string{"--filename", "main.go"})
	command.SetIn(strings.NewReader("ignored terminal input"))
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.input.Filename != "main.go" || service.input.Contents != "package main\n" {
		t.Fatalf("CreateString() input = %+v", service.input)
	}
	pathBytes, err := os.ReadFile(pathLog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(pathBytes)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("submitted draft still exists: %v", err)
	}
}

func TestStringCreateReadsNonterminalInputWithoutEditor(t *testing.T) {
	originalIsTerminalInput := isTerminalInput
	t.Cleanup(func() { isTerminalInput = originalIsTerminalInput })
	isTerminalInput = func(io.Reader) bool { return false }
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
	service := &testStringCreateService{}
	command := newStringCreateCommand(service)
	command.SetArgs([]string{"--filename", "stdin.md"})
	command.SetIn(strings.NewReader("piped contents"))
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.input.Contents != "piped contents" {
		t.Fatalf("contents = %q", service.input.Contents)
	}
}

func TestStringCreateExplicitInputsDoNotOpenEditorForTerminalInput(t *testing.T) {
	originalIsTerminalInput := isTerminalInput
	t.Cleanup(func() { isTerminalInput = originalIsTerminalInput })
	isTerminalInput = func(io.Reader) bool { return true }
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))

	t.Run("standard input", func(t *testing.T) {
		service := &testStringCreateService{}
		command := newStringCreateCommand(service)
		command.SetArgs([]string{"-", "--filename", "stdin.md"})
		command.SetIn(strings.NewReader("standard input contents"))
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)

		if err := command.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if service.input.Filename != "stdin.md" || service.input.Contents != "standard input contents" {
			t.Fatalf("CreateString() input = %+v", service.input)
		}
	})

	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.md")
		if err := os.WriteFile(path, []byte("file contents"), 0o600); err != nil {
			t.Fatal(err)
		}
		service := &testStringCreateService{}
		command := newStringCreateCommand(service)
		command.SetArgs([]string{path})
		command.SetIn(strings.NewReader("ignored terminal input"))
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)

		if err := command.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if service.input.Filename != "file.md" || service.input.Contents != "file contents" {
			t.Fatalf("CreateString() input = %+v", service.input)
		}
	})
}

func TestStringCreateRetainsDraftWhenSubmissionFails(t *testing.T) {
	originalIsTerminalInput := isTerminalInput
	t.Cleanup(func() { isTerminalInput = originalIsTerminalInput })
	isTerminalInput = func(io.Reader) bool { return true }
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "unsubmitted contents\n", 0))
	service := &testStringCreateService{createError: errors.New("network unavailable")}
	command := newStringCreateCommand(service)
	command.SetArgs([]string{"--filename", "draft.txt"})
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

func TestStringCreateRequiresFilenameBeforeOpeningEditor(t *testing.T) {
	originalIsTerminalInput := isTerminalInput
	t.Cleanup(func() { isTerminalInput = originalIsTerminalInput })
	isTerminalInput = func(io.Reader) bool { return true }
	t.Setenv("EDITOR", filepath.Join(t.TempDir(), "missing-editor"))
	command := newStringCreateCommand(nil)
	command.SetErr(io.Discard)

	err := command.Execute()
	if err == nil || err.Error() != "provide --filename when composing in an editor" {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestStringCreateCancelsEmptyEditorDraft(t *testing.T) {
	originalIsTerminalInput := isTerminalInput
	t.Cleanup(func() { isTerminalInput = originalIsTerminalInput })
	isTerminalInput = func(io.Reader) bool { return true }
	pathLog := filepath.Join(t.TempDir(), "path")
	t.Setenv("EDITOR", writeDraftEditor(t, pathLog, "", 0))
	service := &testStringCreateService{}
	command := newStringCreateCommand(service)
	command.SetArgs([]string{"--filename", "empty.txt"})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if service.calls != 0 {
		t.Fatalf("CreateString() calls = %d, want 0", service.calls)
	}
	pathBytes, err := os.ReadFile(pathLog)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(pathBytes)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled draft still exists: %v", err)
	}
}

type testStringCreateService struct {
	createError error
	input       app.CreateStringInput
	calls       int
}

func (service *testStringCreateService) CreateString(_ context.Context, input app.CreateStringInput) (*app.CreatedRecordResult, error) {
	service.calls++
	service.input = input
	if service.createError != nil {
		return nil, service.createError
	}
	return &app.CreatedRecordResult{URI: "at://did:plc:owner/sh.tangled.string/123"}, nil
}
