package gitutil

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	for _, variable := range []string{
		"GIT_CONFIG",
		"GIT_CONFIG_PARAMETERS",
		"GIT_CONFIG_SYSTEM",
	} {
		if err := os.Unsetenv(variable); err != nil {
			panic(fmt.Errorf("unset %s: %w", variable, err))
		}
	}

	settings := []struct {
		name  string
		value string
	}{
		{name: "GIT_CONFIG_COUNT", value: "0"},
		{name: "GIT_CONFIG_GLOBAL", value: os.DevNull},
		{name: "GIT_CONFIG_NOSYSTEM", value: "1"},
	}
	for _, setting := range settings {
		if err := os.Setenv(setting.name, setting.value); err != nil {
			panic(fmt.Errorf("set %s: %w", setting.name, err))
		}
	}

	os.Exit(m.Run())
}

func TestGenerateAndCheckoutPatch(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	originDir := filepath.Join(tempDir, "origin.git")
	repoDir := filepath.Join(tempDir, "repo")
	runGit(t, tempDir, "init", "--bare", originDir)
	runGit(t, tempDir, "clone", originDir, repoDir)
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "switch", "-c", "main")

	writeTestFile(t, filepath.Join(repoDir, "message.txt"), "base\n")
	runGit(t, repoDir, "add", "message.txt")
	runGit(t, repoDir, "commit", "-m", "base")
	runGit(t, repoDir, "push", "-u", "origin", "main")

	runGit(t, repoDir, "switch", "-c", "feature")
	writeTestFile(t, filepath.Join(repoDir, "message.txt"), "feature\n")
	runGit(t, repoDir, "add", "message.txt")
	runGit(t, repoDir, "commit", "-m", "feature")
	compressedPatch, err := GeneratePatch(ctx, repoDir, "main", "feature")
	if err != nil {
		t.Fatalf("GeneratePatch() error = %v", err)
	}
	patch := decompressTestPatch(t, compressedPatch)

	err = CheckoutPatch(ctx, CheckoutPatchParams{
		RepoDir:      repoDir,
		Branch:       "review",
		TargetBranch: "main",
		Patch:        patch,
	})
	if err != nil {
		t.Fatalf("CheckoutPatch() error = %v", err)
	}
	if branch := gitOutputForTest(t, repoDir, "branch", "--show-current"); branch != "review" {
		t.Fatalf("checked out branch = %q, want review", branch)
	}
	contents, err := os.ReadFile(filepath.Join(repoDir, "message.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "feature\n" {
		t.Fatalf("checked out contents = %q, want feature", contents)
	}
}

func TestGeneratePatchPrefersExplicitLocalBaseOverOrigin(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	originDir := filepath.Join(tempDir, "origin.git")
	repoDir := filepath.Join(tempDir, "repo")
	runGit(t, tempDir, "init", "--bare", originDir)
	runGit(t, tempDir, "clone", originDir, repoDir)
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "switch", "-c", "main")
	commitTestFile(t, repoDir, "root.txt", "root\n", "root")
	runGit(t, repoDir, "push", "-u", "origin", "main")
	commitTestFile(t, repoDir, "base.txt", "base\n", "local base")
	runGit(t, repoDir, "switch", "-c", "feature")
	commitTestFile(t, repoDir, "feature.txt", "feature\n", "feature")

	compressedPatch, err := GeneratePatch(ctx, repoDir, "main", "feature")
	if err != nil {
		t.Fatalf("GeneratePatch() error = %v", err)
	}
	patch := string(decompressTestPatch(t, compressedPatch))
	if !strings.Contains(patch, "Subject: [PATCH] feature") {
		t.Fatalf("patch does not contain only the feature commit:\n%s", patch)
	}
	if strings.Contains(patch, "local base") {
		t.Fatalf("patch unexpectedly contains the local base commit:\n%s", patch)
	}
}

func TestCheckoutPatchRejectsDirtyWorktree(t *testing.T) {
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	writeTestFile(t, filepath.Join(repoDir, "untracked.txt"), "dirty\n")

	err := CheckoutPatch(context.Background(), CheckoutPatchParams{RepoDir: repoDir})
	if err == nil || !strings.Contains(err.Error(), "uncommitted changes") {
		t.Fatalf("CheckoutPatch() error = %v, want uncommitted changes error", err)
	}
}

func TestValidateBranchDoesNotStreamStdout(t *testing.T) {
	var stdout bytes.Buffer
	client := NewClient(&stdout, io.Discard)

	if err := client.validateBranch(context.Background(), t.TempDir(), "review"); err != nil {
		t.Fatalf("validateBranch() error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("validateBranch() streamed stdout %q", stdout.String())
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func gitOutputForTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func decompressTestPatch(t *testing.T, compressed []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	patch, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return patch
}
