package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var (
	prCommentBody     string
	prCommentBodyFile string
	prCommentRepo     string
)

var prCommentCmd = &cobra.Command{
	Use:   "comment <rkey>",
	Short: "Add a comment to a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := commandBody(prCommentBody, prCommentBodyFile)
		if err != nil {
			return err
		}
		if body == "" {
			return fmt.Errorf("set --body or --body-file")
		}
		ctx := cmd.Context()
		targetArgs := []string{}
		if prCommentRepo != "" {
			targetArgs = []string{prCommentRepo}
		}
		handle, name, err := resolveTarget(ctx, targetArgs)
		if err != nil {
			return err
		}
		repoDid, err := findRepoDid(ctx, handle, name)
		if err != nil {
			return err
		}
		pulls, err := client.ListPulls(ctx, repoDid, tangled.ListOpts{Limit: defaultListLimit})
		if err != nil {
			return fmt.Errorf("list PRs for %s/%s: %w", handle, name, err)
		}
		pull, err := findByRKey(pulls.Items, args[0], "pull request")
		if err != nil {
			return err
		}

		result, err := createPullComment(ctx, pull.URI, body)
		if err != nil {
			return err
		}
		return output(result, func(result createdRecordResult) {
			fmt.Printf("Added comment %s\n", result.URI)
		})
	},
}

func init() {
	prCommentCmd.Flags().StringVarP(&prCommentBody, "body", "b", "", "Comment body")
	prCommentCmd.Flags().StringVarP(&prCommentBodyFile, "body-file", "F", "", "Read comment body from file")
	prCommentCmd.Flags().StringVarP(&prCommentRepo, "repo", "R", "", "Target repository as handle/repo")
}
