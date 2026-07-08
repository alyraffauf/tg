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

		result := prViewResult{
			Rkey:         rkey,
			Title:        pr.Title,
			Body:         pr.Body,
			Author:       resolveAuthor(ctx, authorDID),
			CreatedAt:    pr.CreatedAt,
			SourceBranch: pr.Source.Branch,
			TargetBranch: pr.Target.Branch,
		}
		return output(result, func(view prViewResult) {
			fmt.Printf("Title:   %s\n", view.Title)
			fmt.Printf("Author:  %s\n", view.Author.Handle)
			fmt.Printf("Created: %s\n", view.CreatedAt)
			fmt.Printf("Branch:  %s → %s\n", view.SourceBranch, view.TargetBranch)
			if view.Body != "" {
				fmt.Printf("\n%s\n", view.Body)
			}
		})
	},
}

func init() {
	prViewCmd.Flags().StringVarP(&prViewRepo, "repo", "R", "", "Target repository as handle/repo")
}
