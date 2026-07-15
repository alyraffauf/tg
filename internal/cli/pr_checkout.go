package cli

import (
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
	Long:  "Check out the latest pull request round on the current remote target branch.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rkey := args[0]
		repoDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
		localRepo, err := gitutil.DetectRepoFromCWD(ctx)
		if err != nil {
			return fmt.Errorf("detect local repository: %w", err)
		}
		localRecord, err := resolveRepoRecord(ctx, localRepo.Handle, localRepo.Repo)
		if err != nil {
			return err
		}

		handle, repoName := localRepo.Handle, localRepo.Repo
		if prCheckoutRepo != "" {
			handle, repoName, err = parseHandleRepo(prCheckoutRepo)
			if err != nil {
				return err
			}
		}
		targetRecord := localRecord
		if handle != localRepo.Handle || repoName != localRepo.Repo {
			targetRecord, err = resolveRepoRecord(ctx, handle, repoName)
			if err != nil {
				return err
			}
		}
		if targetRecord.Value.RepoDid != localRecord.Value.RepoDid {
			return fmt.Errorf("pull request target %s/%s does not match the current repository", handle, repoName)
		}

		pulls, err := client.ListPulls(ctx, targetRecord.Value.RepoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return fmt.Errorf("list PRs for %s/%s: %w", handle, repoName, err)
		}
		pull, err := findByRKey(pulls.Items, rkey, "pull request")
		if err != nil {
			return err
		}
		record, patchCID, err := latestPullPatch(pull, rkey)
		if err != nil {
			return err
		}
		if record.Target.Branch == "" {
			return fmt.Errorf("pull request %q has no target branch", rkey)
		}

		patch, err := downloadPullPatch(ctx, extractDID(pull.URI), patchCID)
		if err != nil {
			return err
		}
		branch := prCheckoutBranch
		if branch == "" {
			branch = "pr-" + rkey
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
		result := prCheckoutResult{Rkey: rkey, Branch: branch}
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
