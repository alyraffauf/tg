package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

const defaultListLimit int64 = 100

// listRow is display-ready data for one issue or PR.
type listRow struct {
	title   string
	state   string
	author  string
	updated string
}

// resolveAuthor returns the handle for didStr, falling back to the
// raw DID string on resolution failure.
func resolveAuthor(ctx context.Context, didStr string) string {
	if ident, err := resolver.ResolveDID(ctx, didStr); err == nil {
		return ident.Handle.String()
	}
	return didStr
}

// shortDate trims an ISO 8601 timestamp to its YYYY-MM-DD prefix.
func shortDate(timestamp string) string {
	if len(timestamp) > 10 {
		return timestamp[:10]
	}
	return timestamp
}

// renderRows writes a table of rows to stdout. emptyMessage is shown
// when rows has no entries.
func renderRows(rows []listRow, emptyMessage string) {
	if len(rows) == 0 {
		fmt.Println(emptyMessage)
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, "TITLE\tSTATE\tAUTHOR\tUPDATED")

	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", row.title, row.state, row.author, row.updated)
	}
	tw.Flush()
}

func extractDID(uri string) string {
	uri = strings.TrimPrefix(uri, "at://")
	did, _, _ := strings.Cut(uri, "/")
	return did
}
