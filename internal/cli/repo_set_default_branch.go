package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/knot"
	"github.com/spf13/cobra"
)

var repoSetDefaultBranchCmd = &cobra.Command{
	Use:   "set-default-branch <branch> [handle/repo]",
	Short: "Set a Tangled repository's default branch",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}
		branch := args[0]
		targetArgs := args[1:]
		handle, name, err := resolveTarget(ctx, targetArgs)
		if err != nil {
			return err
		}
		repo, err := requireOwnedRepo(ctx, handle, name, did)
		if err != nil {
			return err
		}
		if repo.Value.Knot == "" {
			return fmt.Errorf("repo %q has no knot", handle+"/"+name)
		}

		token, err := atClient.GetServiceAuth(ctx, "did:web:"+repo.Value.Knot, "sh.tangled.repo.setDefaultBranch")
		if err != nil {
			return fmt.Errorf("get knot authorization: %w", err)
		}
		if err := knot.New(repo.Value.Knot, token).SetDefaultBranch(ctx, knot.SetDefaultBranchInput{
			Repo:          repo.URI,
			DefaultBranch: branch,
		}); err != nil {
			return err
		}
		return output(repoDefaultBranchResult{URI: repo.URI, Branch: branch}, func(result repoDefaultBranchResult) {
			fmt.Printf("Set default branch for %s to %s\n", result.URI, result.Branch)
		})
	},
}

type repoDefaultBranchResult struct {
	URI    string `json:"uri"`
	Branch string `json:"branch"`
}
