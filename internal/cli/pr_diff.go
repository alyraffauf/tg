package cli

import (
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
		return printPullPatch(ctx, extractDID(pull.URI), cid)
	},
}

func init() {
	prDiffCmd.Flags().StringVarP(&prDiffRepo, "repo", "R", "", "Target repository as handle/repo")
}

func printPullPatch(ctx context.Context, authorDID, cid string) error {
	pdsHost, err := resolver.ResolvePDS(ctx, authorDID)
	if err != nil {
		return fmt.Errorf("resolve PDS for author %q: %w", authorDID, err)
	}
	url := fmt.Sprintf("%s/xrpc/com.atproto.sync.getBlob?did=%s&cid=%s", pdsHost, authorDID, cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build patch download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download patch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download patch: PDS returned HTTP %d", resp.StatusCode)
	}

	patch, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("decompress patch: %w", err)
	}
	defer patch.Close()
	if _, err := io.Copy(os.Stdout, patch); err != nil {
		return fmt.Errorf("write patch: %w", err)
	}
	return nil
}
