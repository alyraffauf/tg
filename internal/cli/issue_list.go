package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var issueListCmd = &cobra.Command{
	Use:   "list [handle/repo]",
	Short: "List issues for a Tangled repository",
	Long: `List issues for a Tangled repository.

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

		issues, err := client.ListIssues(ctx, repoDid, tangled.IssueListOpts{
			Limit: defaultListLimit,
		})
		if err != nil {
			return fmt.Errorf("list issues for %q: %w", repo, err)
		}

		rows := buildIssueRows(ctx, issues.Items)
		renderRows(rows, "No issues found.")
		return nil
	},
}

func parseHandleRepo(arg string) (string, string, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected handle/repo, got %q", arg)
	}
	return parts[0], parts[1], nil
}

// resolveTarget returns the handle and repo name from an explicit
// "handle/repo" argument or by detecting the git remote in the CWD.
func resolveTarget(ctx context.Context, args []string) (string, string, error) {
	if len(args) == 1 {
		return parseHandleRepo(args[0])
	}

	rc, err := gitutil.DetectRepoFromCWD(ctx)
	if err != nil {
		return "", "", fmt.Errorf("detect repo from current directory: %w", err)
	}
	return rc.Handle, rc.Repo, nil
}

// findRepoDid resolves handle/repo to the repo's repoDid, which listIssues is
// keyed by. It looks the record up directly by name (current schema uses the
// name as the rkey), falling back to a listing for legacy repos whose rkey is a
// TID with the name in the body.
func findRepoDid(ctx context.Context, handle, repo string) (string, error) {
	ident, err := resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return "", fmt.Errorf("resolve handle %q: %w", handle, err)
	}

	repoURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, repo)
	if got, err := client.GetRepo(ctx, repoURI); err == nil {
		return got.Value.RepoDid, nil
	}

	if repos, err := client.ListRepos(ctx, ident.DID.String()); err == nil {
		for _, item := range repos.Items {
			if item.Value.Name == repo || strings.HasSuffix(item.URI, "/"+repo) {
				return item.Value.RepoDid, nil
			}
		}
	}

	return "", fmt.Errorf("repo %q not found for handle %q", repo, handle)
}

// buildIssueRows resolves each issue author's DID to a handle, falling
// back to the raw DID on resolution failure.
func buildIssueRows(ctx context.Context, items []tangled.IssueListItem) []listRow {
	rows := make([]listRow, 0, len(items))

	for _, item := range items {
		var record tangled.IssueRecord
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
			rkey:    extractRKey(item.URI),
			title:   title,
			state:   item.State,
			author:  resolveAuthor(ctx, extractDID(item.URI)),
			updated: shortDate(updated),
		})
	}

	return rows
}
