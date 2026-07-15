package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var (
	prCheckoutRepo   string
	prCheckoutBranch string
	prCheckoutForce  bool
)

var prCheckoutCmd = &cobra.Command{
	Use:   "checkout <rkey>",
	Short: "Check out a pull request in Git",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		repoDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
		local, err := gitutil.DetectRepoFromCWD(ctx)
		if err != nil {
			return fmt.Errorf("detect local repository: %w", err)
		}
		localRecord, err := resolveRepoRecord(ctx, local.Handle, local.Repo)
		if err != nil {
			return err
		}

		handle, name := local.Handle, local.Repo
		if prCheckoutRepo != "" {
			handle, name, err = parseHandleRepo(prCheckoutRepo)
			if err != nil {
				return err
			}
		}
		targetRecord := localRecord
		if handle != local.Handle || name != local.Repo {
			targetRecord, err = resolveRepoRecord(ctx, handle, name)
			if err != nil {
				return err
			}
		}
		if targetRecord.Value.RepoDid != localRecord.Value.RepoDid {
			return fmt.Errorf("pull request target %s/%s does not match the current repository", handle, name)
		}

		pulls, err := client.ListPulls(ctx, targetRecord.Value.RepoDid, tangled.ListOpts{Limit: defaultListLimit})
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
		if record.Target.Branch == "" {
			return fmt.Errorf("pull request %q has no target branch", args[0])
		}

		latestRound := record.Rounds[len(record.Rounds)-1]
		cid := latestRound.PatchBlob.Ref.String()
		if cid == "" {
			return fmt.Errorf("pull request %q has no patch blob", args[0])
		}
		patch, err := downloadPullPatch(ctx, extractDID(pull.URI), cid)
		if err != nil {
			return err
		}
		branch := prCheckoutBranch
		if branch == "" {
			branch = "pr-" + args[0]
		}

		if err := gitutil.CheckoutPatch(ctx, gitutil.CheckoutPatchParams{
			RepoDir:      repoDir,
			Branch:       branch,
			TargetBranch: record.Target.Branch,
			Patch:        patch,
			Force:        prCheckoutForce,
		}); err != nil {
			return err
		}
		result := prCheckoutResult{Rkey: args[0], Branch: branch}
		return output(result, func(result prCheckoutResult) {
			fmt.Printf("Checked out pull request %s as branch %s\n", result.Rkey, result.Branch)
		})
	},
}

type prCheckoutResult struct {
	Rkey   string `json:"rkey"`
	Branch string `json:"branch"`
}

func init() {
	prCheckoutCmd.Flags().StringVarP(&prCheckoutRepo, "repo", "R", "", "Target repository as handle/repo")
	prCheckoutCmd.Flags().StringVarP(&prCheckoutBranch, "branch", "b", "", "Local branch name (default: pr-<rkey>)")
	prCheckoutCmd.Flags().BoolVarP(&prCheckoutForce, "force", "f", false, "Reset an existing checkout branch")
}
