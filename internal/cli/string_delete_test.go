package cli

import (
	"testing"
)

func TestStringDeleteCmd(t *testing.T) {
	command := newStringDeleteCommand(nil)
	if command == nil {
		t.Fatal("stringDeleteCmd is nil")
	}
	if command.Use != "delete <rkey>" {
		t.Errorf("Use = %q, want %q", command.Use, "delete <rkey>")
	}
	// cobra.ExactArgs(1): zero args must error, one arg must succeed.
	if err := command.Args(nil, []string{}); err == nil {
		t.Error("expected error for zero args, got nil")
	}
	if err := command.Args(nil, []string{"3k2abc"}); err != nil {
		t.Errorf("expected no error for one arg, got %v", err)
	}
	if err := command.Args(nil, []string{"a", "b"}); err == nil {
		t.Error("expected error for two args, got nil")
	}
}
