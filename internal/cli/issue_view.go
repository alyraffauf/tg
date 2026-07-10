package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var issueViewRepo string

var issueViewCmd = &cobra.Command{
	Use:   "view <rkey>",
	Short: "View an issue for a Tangled repository",
	Long: `View an issue by its rkey (the last segment of its at:// URI).

If --repo is not set, the repository is detected from the current
directory's git origin remote.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rkey := args[0]

		targetArgs := []string{}
		if issueViewRepo != "" {
			targetArgs = []string{issueViewRepo}
		}
		handle, repo, err := resolveTarget(ctx, targetArgs)
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
			return fmt.Errorf("list issues for %s/%s: %w", handle, repo, err)
		}

		found, err := findByRKey(issues.Items, rkey, "issue")
		if err != nil {
			return err
		}
		decoded, err := decodeIssue(found.Value)
		if err != nil {
			return fmt.Errorf("decode issue %q: %w", rkey, err)
		}

		result := viewResult{
			Rkey:      rkey,
			Title:     decoded.Title,
			Body:      decoded.Body,
			Author:    resolveAuthor(ctx, extractDID(found.URI)),
			CreatedAt: decoded.CreatedAt,
		}
		return output(result, func(view viewResult) {
			fmt.Printf("Title:   %s\n", view.Title)
			fmt.Printf("Author:  %s\n", view.Author.Handle)
			fmt.Printf("Created: %s\n", view.CreatedAt)
			if view.Body != "" {
				fmt.Printf("\n%s\n", view.Body)
			}
		})
	},
}

func init() {
	issueViewCmd.Flags().StringVarP(&issueViewRepo, "repo", "R", "", "Target repository as handle/repo")
}
