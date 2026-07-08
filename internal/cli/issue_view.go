package cli

import (
	"encoding/json"
	"fmt"
	"strings"

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

		issues, err := client.ListIssues(ctx, repoDid, tangled.IssueListOpts{
			Limit: defaultListLimit,
		})
		if err != nil {
			return fmt.Errorf("list issues for %s/%s: %w", handle, repo, err)
		}

		issue, authorDID, err := findIssueByRKey(issues.Items, rkey)
		if err != nil {
			return err
		}

		fmt.Printf("Title:   %s\n", issue.Title)
		fmt.Printf("Author:  %s\n", resolveAuthor(ctx, authorDID))
		fmt.Printf("Created: %s\n", issue.CreatedAt)
		if issue.Body != "" {
			fmt.Printf("\n%s\n", issue.Body)
		}
		return nil
	},
}

func init() {
	issueViewCmd.Flags().StringVarP(&issueViewRepo, "repo", "R", "", "Target repository as handle/repo")
}

func findIssueByRKey(items []tangled.IssueListItem, rkey string) (*tangled.IssueRecord, string, error) {
	for _, item := range items {
		if !strings.HasSuffix(item.URI, "/"+rkey) {
			continue
		}
		var issue tangled.IssueRecord
		if err := json.Unmarshal(item.Value, &issue); err != nil {
			return nil, "", fmt.Errorf("decode issue %q: %w", rkey, err)
		}
		return &issue, extractDID(item.URI), nil
	}
	return nil, "", fmt.Errorf("issue %q not found", rkey)
}
