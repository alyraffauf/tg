package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func TestOutputJSONSerializesWarningsWithoutHumanOutput(t *testing.T) {
	var stdout bytes.Buffer
	command := &cobra.Command{}
	command.Flags().Bool("json", false, "")
	if err := command.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	command.SetOut(&stdout)
	humanCalled := false
	result := &app.RepoListResult{
		Items:    []app.RepoItem{{Name: "valid"}},
		Warnings: []app.RecordWarning{{URI: "at://bad", Error: "malformed"}},
	}
	if err := output(command, result, func(*app.RepoListResult) { humanCalled = true }); err != nil {
		t.Fatalf("output() error = %v", err)
	}
	if humanCalled {
		t.Fatal("human renderer called in JSON mode")
	}
	want := "{\n  \"items\": [\n    {\n      \"name\": \"valid\",\n      \"uri\": \"\",\n      \"author\": \"\",\n      \"knot\": \"\",\n      \"createdAt\": \"\"\n    }\n  ],\n  \"warnings\": [\n    {\n      \"uri\": \"at://bad\",\n      \"error\": \"malformed\"\n    }\n  ]\n}\n"
	if stdout.String() != want {
		t.Fatalf("JSON output:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

func TestOutputJSONPropagatesWriterFailure(t *testing.T) {
	command := &cobra.Command{}
	command.Flags().Bool("json", true, "")
	command.SetOut(errorWriter{})
	if err := output(command, map[string]string{"key": "value"}, func(map[string]string) {}); !errors.Is(err, errWrite) {
		t.Fatalf("output() error = %v", err)
	}
}

var errWrite = errors.New("write failed")

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errWrite }
