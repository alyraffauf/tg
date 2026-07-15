package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var (
	issueCommentBody     string
	issueCommentBodyFile string
	issueCommentRepo     string
)

var issueCommentCmd = &cobra.Command{
	Use:   "comment <rkey>",
	Short: "Add a comment to an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := commandBody(issueCommentBody, issueCommentBodyFile)
		if err != nil {
			return err
		}
		if body == "" {
			return fmt.Errorf("set --body or --body-file")
		}
		ctx := cmd.Context()
		targetArgs := []string{}
		if issueCommentRepo != "" {
			targetArgs = []string{issueCommentRepo}
		}
		handle, name, err := resolveTarget(ctx, targetArgs)
		if err != nil {
			return err
		}
		repoDid, err := findRepoDid(ctx, handle, name)
		if err != nil {
			return err
		}
		issues, err := client.ListIssues(ctx, repoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return fmt.Errorf("list issues for %s/%s: %w", handle, name, err)
		}
		issue, err := findByRKey(issues.Items, args[0], "issue")
		if err != nil {
			return err
		}

		result, err := createIssueComment(ctx, issue.URI, body)
		if err != nil {
			return err
		}
		return output(result, func(result createdRecordResult) {
			fmt.Printf("Added comment %s\n", result.URI)
		})
	},
}

func init() {
	issueCommentCmd.Flags().StringVarP(&issueCommentBody, "body", "b", "", "Comment body")
	issueCommentCmd.Flags().StringVarP(&issueCommentBodyFile, "body-file", "F", "", "Read comment body from file")
	issueCommentCmd.Flags().StringVarP(&issueCommentRepo, "repo", "R", "", "Target repository as handle/repo")
}
