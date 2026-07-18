package cli

import (
	"testing"
)

func TestStringDeleteCmd(t *testing.T) {
	if stringDeleteCmd == nil {
		t.Fatal("stringDeleteCmd is nil")
	}
	if stringDeleteCmd.Use != "delete <rkey>" {
		t.Errorf("Use = %q, want %q", stringDeleteCmd.Use, "delete <rkey>")
	}
	// cobra.ExactArgs(1): zero args must error, one arg must succeed.
	if err := stringDeleteCmd.Args(nil, []string{}); err == nil {
		t.Error("expected error for zero args, got nil")
	}
	if err := stringDeleteCmd.Args(nil, []string{"3k2abc"}); err != nil {
		t.Errorf("expected no error for one arg, got %v", err)
	}
	if err := stringDeleteCmd.Args(nil, []string{"a", "b"}); err == nil {
		t.Error("expected error for two args, got nil")
	}
}
