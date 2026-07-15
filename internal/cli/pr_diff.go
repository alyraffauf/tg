package cli

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

const maxPullPatchSize = 100 << 20

var prDiffRepo string

var prDiffCmd = &cobra.Command{
	Use:   "diff <rkey>",
	Short: "Print the latest patch for a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		targetArgs := []string{}
		if prDiffRepo != "" {
			targetArgs = []string{prDiffRepo}
		}
		handle, name, err := resolveTarget(ctx, targetArgs)
		if err != nil {
			return err
		}
		repoDid, err := findRepoDid(ctx, handle, name)
		if err != nil {
			return err
		}
		pulls, err := client.ListPulls(ctx, repoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return fmt.Errorf("list PRs for %s/%s: %w", handle, name, err)
		}
		pull, err := findByRKey(pulls.Items, args[0], "pull request")
		if err != nil {
			return err
		}
		var record tangled.PullRecord
		if err := json.Unmarshal(pull.Value, &record); err != nil {
			return fmt.Errorf("decode pull request %q: %w", args[0], err)
		}
		if len(record.Rounds) == 0 {
			return fmt.Errorf("pull request %q has no rounds", args[0])
		}
		cid := record.Rounds[len(record.Rounds)-1].PatchBlob.Ref.String()
		if cid == "" {
			return fmt.Errorf("pull request %q has no patch blob", args[0])
		}
		patch, err := downloadPullPatch(ctx, extractDID(pull.URI), cid)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(patch)
		return err
	},
}

func init() {
	prDiffCmd.Flags().StringVarP(&prDiffRepo, "repo", "R", "", "Target repository as handle/repo")
}

func downloadPullPatch(ctx context.Context, authorDID, cid string) ([]byte, error) {
	pdsHost, err := resolver.ResolvePDS(ctx, authorDID)
	if err != nil {
		return nil, fmt.Errorf("resolve PDS for author %q: %w", authorDID, err)
	}
	url := fmt.Sprintf("%s/xrpc/com.atproto.sync.getBlob?did=%s&cid=%s", pdsHost, authorDID, cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build patch download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download patch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download patch: PDS returned HTTP %d", resp.StatusCode)
	}

	compressed, err := readLimited(resp.Body, maxPullPatchSize)
	if err != nil {
		return nil, fmt.Errorf("download patch: %w", err)
	}
	patch, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("decompress patch: %w", err)
	}
	defer patch.Close()
	contents, err := readLimited(patch, maxPullPatchSize)
	if err != nil {
		return nil, fmt.Errorf("decompress patch: %w", err)
	}
	return contents, nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	contents, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(contents)) > limit {
		return nil, fmt.Errorf("patch exceeds %d bytes", limit)
	}
	return contents, nil
}
