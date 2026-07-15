package cli

import (
	"fmt"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

var (
	issueCreateBody     string
	issueCreateBodyFile string
	issueCreateRepo     string
)

var issueCreateCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create an issue on a Tangled repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		body, err := commandBody(issueCreateBody, issueCreateBodyFile)
		if err != nil {
			return err
		}
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}

		targetArgs := []string{}
		if issueCreateRepo != "" {
			targetArgs = []string{issueCreateRepo}
		}
		handle, name, err := resolveTarget(ctx, targetArgs)
		if err != nil {
			return err
		}
		repoDid, err := findRepoDid(ctx, handle, name)
		if err != nil {
			return err
		}

		rkey := string(syntax.NewTIDNow(0))
		uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
			Repo:       did,
			Collection: "sh.tangled.repo.issue",
			Rkey:       rkey,
			Record: tangled.IssueRecord{
				Type:      "sh.tangled.repo.issue",
				Repo:      repoDid,
				Title:     args[0],
				Body:      body,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		})
		if err != nil {
			return fmt.Errorf("create issue: %w", err)
		}

		return output(createdRecordResult{Rkey: rkey, URI: uri}, func(result createdRecordResult) {
			fmt.Printf("Created issue %s\n", result.URI)
		})
	},
}

func init() {
	issueCreateCmd.Flags().StringVarP(&issueCreateBody, "body", "b", "", "Issue body")
	issueCreateCmd.Flags().StringVarP(&issueCreateBodyFile, "body-file", "F", "", "Read issue body from file")
	issueCreateCmd.Flags().StringVarP(&issueCreateRepo, "repo", "R", "", "Target repository as handle/repo")
}
