package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var prCheckoutCmd = &cobra.Command{
	Use:   "checkout <pr-rkey>",
	Short: "Check out a pull request as a detached HEAD",
	Long: `Check out a pull request by fetching its target branch and applying
the PR's gzipped patch blob on top, leaving you in a detached HEAD.

Must be run from inside a cloned Tangled repository.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prRKey := args[0]

		// No explicit handle/repo: must auto-detect from CWD.
		handle, repo, err := resolveTarget(ctx, nil)
		if err != nil {
			return err
		}

		repoDid, err := findRepoDid(ctx, handle, repo)
		if err != nil {
			return err
		}

		pulls, err := client.ListPulls(ctx, repoDid, tangled.ListOpts{
			Limit: defaultListLimit,
		})
		if err != nil {
			return fmt.Errorf("list pulls for %q: %w", repo, err)
		}

		found, err := findByRKey(pulls.Items, prRKey, "pull request")
		if err != nil {
			return err
		}
		var pr tangled.PullRecord
		if err := json.Unmarshal(found.Value, &pr); err != nil {
			return fmt.Errorf("decode pull request %q: %w", prRKey, err)
		}
		authorDID := extractDID(found.URI)

		if len(pr.Rounds) == 0 {
			return fmt.Errorf("pull request %q has no rounds", prRKey)
		}

		// The last round is the latest patch revision.
		cid := pr.Rounds[len(pr.Rounds)-1].PatchBlob.Ref.String()
		if cid == "" {
			return fmt.Errorf("pull request %q has no patch blob", prRKey)
		}

		pdsHost, err := resolver.ResolvePDS(ctx, authorDID)
		if err != nil {
			return fmt.Errorf("resolve PDS for author %q: %w", authorDID, err)
		}

		repoDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}

		if err := gitutil.CheckoutPull(ctx, gitutil.CheckoutPullParams{
			RepoDir:      repoDir,
			PDSHost:      pdsHost,
			AuthorDID:    authorDID,
			CID:          cid,
			TargetHandle: handle,
			TargetRepo:   repo,
			TargetBranch: pr.Target.Branch,
		}); err != nil {
			return fmt.Errorf("checkout pull %q: %w", prRKey, err)
		}

		result := prCheckoutResult{
			Rkey:      prRKey,
			Branch:    pr.Target.Branch,
			Directory: repoDir,
		}
		return output(result, func(checkout prCheckoutResult) {
			fmt.Printf("Checked out PR %s as detached HEAD in %s\n", checkout.Rkey, checkout.Directory)
		})
	},
}
