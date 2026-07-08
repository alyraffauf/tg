package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/spf13/cobra"
)

var repoCloneCmd = &cobra.Command{
	Use:   "clone <handle/repo> [directory]",
	Short: "Clone a Tangled repository",
	Long: `Clone a Tangled repository via SSH into a local directory.

The default destination is the repository name.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		handle, repo, err := parseHandleRepo(args[0])
		if err != nil {
			return err
		}

		dest := repo
		if len(args) == 2 {
			dest = args[1]
		}

		fmt.Printf("Cloning %s/%s into %s...\n", handle, repo, dest)
		if err := gitutil.CloneRepo(ctx, gitutil.CloneRepoParams{
			Handle:  handle,
			Repo:    repo,
			RepoDir: dest,
		}); err != nil {
			return fmt.Errorf("clone %q: %w", args[0], err)
		}
		return nil
	},
}
