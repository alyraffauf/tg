package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/alyraffauf/tg/internal/app"
)

// shortDate trims an ISO 8601 timestamp to its YYYY-MM-DD prefix.
func shortDate(timestamp string) string {
	if len(timestamp) > 10 {
		return timestamp[:10]
	}
	return timestamp
}

// renderTable writes a tab-aligned table of rows to stdout under header.
// emptyMessage is shown when rows has no entries. Every renderer in this
// package (issues, pulls, repos, SSH keys) goes through this.
func renderTable(writer io.Writer, header []string, rows [][]string, emptyMessage string) {
	if len(rows) == 0 {
		fmt.Fprintln(writer, emptyMessage)
		return
	}

	tw := tabwriter.NewWriter(writer, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

// renderList renders issue or pull-request items as a table.
func renderList(writer io.Writer, items []app.Item, emptyMessage string) {
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.Rkey, it.Title, it.State, it.Author.Handle, shortDate(it.UpdatedAt)})
	}
	renderTable(writer, []string{"RKEY", "TITLE", "STATE", "AUTHOR", "UPDATED"}, rows, emptyMessage)
}
