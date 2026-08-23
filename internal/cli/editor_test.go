package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDraftContentsStopsAtLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "draft")
	if err := os.WriteFile(path, []byte("123456789"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := readDraftContents(path, "string", 8)
	if err == nil || !strings.Contains(err.Error(), "contents exceed") || !strings.Contains(err.Error(), path) {
		t.Fatalf("readDraftContents() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("retained draft %q: %v", path, err)
	}
}

func TestEditDraftRemovesUntouchedDraftWhenEditorCannotOpen(t *testing.T) {
	t.Setenv("SNAP_REVISION", "test")
	t.Setenv("TMPDIR", t.TempDir())

	_, err := editDraft(context.Background(), "issue", "tg-issue-*.md", "template", nil, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "open issue editor") {
		t.Fatalf("editDraft() error = %v", err)
	}
	entries, err := os.ReadDir(os.Getenv("TMPDIR"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary directory contains %v after editor setup failure", entries)
	}
}
