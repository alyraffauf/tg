package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var prListCmd = &cobra.Command{
	Use:   "list [handle/repo]",
	Short: "List pull requests for a Tangled repository",
	Long: `List pull requests for a Tangled repository.

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

		pulls, err := client.ListPulls(ctx, repoDid, tangled.PullListOpts{
			Limit: defaultListLimit,
		})
		if err != nil {
			return fmt.Errorf("list PRs for %q: %w", repo, err)
		}

		items := buildPullItems(ctx, pulls.Items)
		return output(items, renderPullList)
	},
}

func buildPullItems(ctx context.Context, items []tangled.PullListItem) []pullItem {
	result := make([]pullItem, 0, len(items))

	for _, item := range items {
		var record tangled.PullRecord
		if err := json.Unmarshal(item.Value, &record); err != nil {
			continue
		}

		updated := item.StateUpdatedAt
		if updated == "" {
			updated = record.CreatedAt
		}

		title := record.Title
		if title == "" {
			title = "(no title)"
		}

		result = append(result, pullItem{
			Rkey:         extractRKey(item.URI),
			URI:          item.URI,
			Title:        title,
			State:        item.State,
			Author:       resolveAuthor(ctx, extractDID(item.URI)),
			CreatedAt:    record.CreatedAt,
			UpdatedAt:    updated,
			CommentCount: item.CommentCount,
			SourceBranch: record.Source.Branch,
			TargetBranch: record.Target.Branch,
		})
	}

	return result
}

func renderPullList(items []pullItem) {
	rows := make([]listRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, listRow{
			rkey:    item.Rkey,
			title:   item.Title,
			state:   item.State,
			author:  item.Author.Handle,
			updated: shortDate(item.UpdatedAt),
		})
	}
	renderRows(rows, "No pull requests found.")
}
