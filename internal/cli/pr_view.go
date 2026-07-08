package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var prViewRepo string

var prViewCmd = &cobra.Command{
	Use:   "view <rkey>",
	Short: "View a pull request for a Tangled repository",
	Long: `View a pull request by its rkey (the last segment of its at:// URI).

If --repo is not set, the repository is detected from the current
directory's git origin remote.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rkey := args[0]

		targetArgs := []string{}
		if prViewRepo != "" {
			targetArgs = []string{prViewRepo}
		}
		handle, repo, err := resolveTarget(ctx, targetArgs)
		if err != nil {
			return err
		}

		repoDid, err := findRepoDid(ctx, handle, repo)
		if err != nil {
			return err
		}

		pulls, err := client.ListPulls(ctx, repoDid, tangled.PullListOpts{
			Limit: defaultListLimit,
		})
		if err != nil {
			return fmt.Errorf("list PRs for %s/%s: %w", handle, repo, err)
		}

		pr, authorDID, err := findPullByRKey(pulls.Items, rkey)
		if err != nil {
			return err
		}

		fmt.Printf("Title:   %s\n", pr.Title)
		fmt.Printf("Author:  %s\n", resolveAuthor(ctx, authorDID))
		fmt.Printf("Created: %s\n", pr.CreatedAt)
		fmt.Printf("Branch:  %s → %s\n", pr.Source.Branch, pr.Target.Branch)
		if pr.Body != "" {
			fmt.Printf("\n%s\n", pr.Body)
		}
		return nil
	},
}

func init() {
	prViewCmd.Flags().StringVarP(&prViewRepo, "repo", "R", "", "Target repository as handle/repo")
}
