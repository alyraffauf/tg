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
	rkey    string
	title   string
	state   string
	author  string
	updated string
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
	fmt.Fprintln(tw, "RKEY\tTITLE\tSTATE\tAUTHOR\tUPDATED")

	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", row.rkey, row.title, row.state, row.author, row.updated)
	}
	tw.Flush()
}

func extractDID(uri string) string {
	uri = strings.TrimPrefix(uri, "at://")
	did, _, _ := strings.Cut(uri, "/")
	return did
}

func extractRKey(uri string) string {
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

// resolveAuthor resolves a DID to an author, falling back to the raw
// DID string for Handle if resolution fails.
func resolveAuthor(ctx context.Context, did string) author {
	result := author{DID: did}
	if ident, err := resolver.ResolveDID(ctx, did); err == nil {
		result.Handle = ident.Handle.String()
	} else {
		result.Handle = did
	}
	return result
}
