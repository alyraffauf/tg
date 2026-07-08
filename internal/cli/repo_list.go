package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var repoListCmd = &cobra.Command{
	Use:   "list [handle]",
	Short: "List repositories owned by a Tangled user",
	Long: `List repositories owned by a Tangled user.

If no argument is given, the command detects the user from the "origin"
remote URL of the git repository in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		handle, err := resolveHandleArg(ctx, args)
		if err != nil {
			return err
		}

		ident, err := resolver.ResolveHandle(ctx, handle)
		if err != nil {
			return fmt.Errorf("resolve handle %q: %w", handle, err)
		}

		repos, err := client.ListRepos(ctx, ident.DID.String())
		if err != nil {
			return fmt.Errorf("list repos for %q: %w", handle, err)
		}

		renderRepos(buildRepoRows(repos.Items))
		return nil
	},
}

// resolveHandleArg returns the handle from an explicit argument, or
// falls back to the handle of the CWD's git origin remote.
func resolveHandleArg(ctx context.Context, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	rc, err := gitutil.DetectRepoFromCWD(ctx)
	if err != nil {
		return "", fmt.Errorf("detect repo from current directory: %w", err)
	}
	return rc.Handle, nil
}

// resolveHandleOrSelf returns the handle from an explicit argument, or the
// authenticated user's handle. It does not fall back to CWD git detection.
func resolveHandleOrSelf(ctx context.Context, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if auth == nil || !auth.IsAuthenticated() {
		return "", fmt.Errorf("not logged in; provide a handle or run \"tg auth login\"")
	}
	ident, err := resolver.ResolveDID(ctx, auth.CurrentDID().String())
	if err != nil {
		return "", fmt.Errorf("resolve your DID: %w", err)
	}
	return ident.Handle.String(), nil
}

type repoRow struct {
	name        string
	knot        string
	description string
	created     string
}

func buildRepoRows(items []tangled.Repo) []repoRow {
	rows := make([]repoRow, 0, len(items))

	for _, item := range items {
		name := item.Value.Name
		if name == "" {
			// Fall back to the rkey from the at:// URI.
			if idx := strings.LastIndex(item.URI, "/"); idx != -1 {
				name = item.URI[idx+1:]
			}
		}

		rows = append(rows, repoRow{
			name:        name,
			knot:        item.Value.Knot,
			description: item.Value.Description,
			created:     shortDate(item.Value.CreatedAt),
		})
	}

	return rows
}

func renderRepos(rows []repoRow) {
	if len(rows) == 0 {
		fmt.Println("No repositories found.")
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, "NAME\tKNOT\tDESCRIPTION\tCREATED")

	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", row.name, row.knot, row.description, row.created)
	}
	tw.Flush()
}
