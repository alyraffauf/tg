package gitutil

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCloneRepoTerminatesGitOptions(t *testing.T) {
	binDir := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "args")
	gitPath := filepath.Join(binDir, "git")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$TG_TEST_ARGS_FILE\"\n"
	if err := os.WriteFile(gitPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake git: %v", err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("TG_TEST_ARGS_FILE", argsFile)

	err := NewClient(nil, nil).CloneRepo(context.Background(), CloneRepoParams{
		Protocol: "https", RepoDID: "did:plc:repository", RepoDir: "--bare",
	})
	if err != nil {
		t.Fatalf("CloneRepo() error = %v", err)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read fake git arguments: %v", err)
	}
	got := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	want := []string{"clone", "--", "https://tangled.org/did:plc:repository", "--bare"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("git arguments = %q, want %q", got, want)
	}
}
