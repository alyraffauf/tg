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

		result := issueViewResult{
			Rkey:      rkey,
			Title:     issue.Title,
			Body:      issue.Body,
			Author:    resolveAuthor(ctx, authorDID),
			CreatedAt: issue.CreatedAt,
		}
		return output(result, func(view issueViewResult) {
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
