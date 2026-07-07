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

		rows := buildPullRows(ctx, pulls.Items)
		renderRows(rows, "No pull requests found.")
		return nil
	},
}

// buildPullRows resolves each PR author's DID to a handle, falling back
// to the raw DID on resolution failure.
func buildPullRows(ctx context.Context, items []tangled.PullListItem) []listRow {
	rows := make([]listRow, 0, len(items))

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

		rows = append(rows, listRow{
			title:   title,
			state:   item.State,
			author:  resolveAuthor(ctx, extractDID(item.URI)),
			updated: shortDate(updated),
		})
	}

	return rows
}
