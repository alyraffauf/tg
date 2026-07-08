package gitutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

// CheckoutPullParams groups the inputs to CheckoutPull.
type CheckoutPullParams struct {
	RepoDir      string // local git repo to operate in
	PDSHost      string // author's PDS base URL
	AuthorDID    string // DID of the PR author
	CID          string // CID of the patch blob
	TargetHandle string // target repo owner handle
	TargetRepo   string // target repo name
	TargetBranch string // target branch to fetch and detach onto
}

// CheckoutPull fetches the target branch as detached HEAD and applies
// the PR's gzipped patch blob on top, all inside params.RepoDir.
func CheckoutPull(ctx context.Context, params CheckoutPullParams) error {
	fetchURL := tangledRemoteURL(params.TargetHandle, params.TargetRepo)
	if err := runIn(params.RepoDir, ctx, "git", "fetch", fetchURL, params.TargetBranch); err != nil {
		return fmt.Errorf("fetch target branch: %w", err)
	}
	if err := runIn(params.RepoDir, ctx, "git", "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("checkout detached HEAD: %w", err)
	}

	blobURL := fmt.Sprintf("%s/xrpc/com.atproto.sync.getBlob?did=%s&cid=%s", params.PDSHost, params.AuthorDID, params.CID)
	req, err := http.NewRequestWithContext(ctx, "GET", blobURL, nil)
	if err != nil {
		return fmt.Errorf("build blob request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download patch blob: %w", err)
	}
	defer resp.Body.Close()

	return applyGzippedPatch(ctx, params.RepoDir, resp.Body)
}

// applyGzippedPatch decompresses a gzipped patch and applies it via
// `git am` inside repoDir.
func applyGzippedPatch(ctx context.Context, repoDir string, body io.Reader) error {
	gunzip := exec.CommandContext(ctx, "gunzip")
	gunzip.Stdin = body

	gitAm := exec.CommandContext(ctx, "git", "am")
	gitAm.Dir = repoDir
	gitAm.Stdout = os.Stdout
	gitAm.Stderr = os.Stderr

	pipe, err := gunzip.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create pipe: %w", err)
	}
	gitAm.Stdin = pipe

	if err := gunzip.Start(); err != nil {
		return fmt.Errorf("start gunzip: %w", err)
	}
	if err := gitAm.Run(); err != nil {
		return fmt.Errorf("git am: %w", err)
	}
	if err := gunzip.Wait(); err != nil {
		return fmt.Errorf("gunzip: %w", err)
	}

	return nil
}
