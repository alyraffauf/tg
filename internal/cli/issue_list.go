package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var issueListCmd = &cobra.Command{
	Use:   "list [handle/repo]",
	Short: "List issues for a Tangled repository",
	Long: `List issues for a Tangled repository.

If no argument is given, the command detects the repository from the
"origin" remote URL of the git repository in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		handle, repo, err := resolveTarget(ctx, args)
		if err != nil {
			return err
		}

		repoDid, err := findRepoDid(ctx, handle, repo)
		if err != nil {
			return err
		}

		issues, err := client.ListIssues(ctx, repoDid, tangled.ListOpts{
			Limit: defaultListLimit,
		})
		if err != nil {
			return fmt.Errorf("list issues for %q: %w", repo, err)
		}

		items := buildItems(ctx, issues.Items, decodeIssue)
		return output(items, func(items []item) {
			renderList(items, "No issues found.")
		})
	},
}
