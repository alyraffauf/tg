package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/alyraffauf/tg/internal/app"
	xterm "github.com/charmbracelet/x/term"
)

// shortDate trims an ISO 8601 timestamp to its YYYY-MM-DD prefix.
func shortDate(timestamp string) string {
	if len(timestamp) > 10 {
		return timestamp[:10]
	}
	return timestamp
}

// asFile returns w as an *os.File, or nil when w is not backed by a file.
func asFile(w io.Writer) *os.File {
	f, ok := w.(*os.File)
	if !ok {
		return nil
	}
	return f
}

// isTerminal reports whether w is an interactive terminal.
var isTerminal = func(w io.Writer) bool {
	f := asFile(w)
	return f != nil && xterm.IsTerminal(f.Fd())
}

// terminalWidth returns the column width of w's terminal, or 0 when w is
// not a terminal or the width can't be determined.
var terminalWidth = func(w io.Writer) int {
	f := asFile(w)
	if f == nil {
		return 0
	}
	if width, _, err := xterm.GetSize(f.Fd()); err == nil {
		return width
	}
	return 0
}

// renderTable writes a table of rows under header to writer. When writer
// is a terminal the table is drawn with a border and a bold header; when
// piped or redirected it falls back to a plain tab-aligned table.
// emptyMessage is shown when rows is empty.
func renderTable(writer io.Writer, header []string, rows [][]string, emptyMessage string) {
	if len(rows) == 0 {
		fmt.Fprintln(writer, emptyMessage)
		return
	}

	if isTerminal(writer) {
		renderBorderedTable(writer, header, rows)
		return
	}

	tw := tabwriter.NewWriter(writer, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

// renderBorderedTable renders a bordered, styled table for interactive
// terminals. Cells truncate (with "…") rather than wrap; on narrow
// terminals the table is shrunk to fit the terminal width.
func renderBorderedTable(writer io.Writer, header []string, rows [][]string) {
	rendered := newBorderedTable(header, rows).Render()

	// The top border spans the full table width, so it tells us whether
	// the natural width fits the terminal. If it overflows, re-render
	// with a width cap so lipgloss shrinks columns instead of spilling.
	if w := terminalWidth(writer); w > 0 {
		if first, _, _ := strings.Cut(rendered, "\n"); lipgloss.Width(first) > w {
			rendered = newBorderedTable(header, rows).Width(w).Render()
		}
	}

	fmt.Fprintln(writer, rendered)
}

func newBorderedTable(header []string, rows [][]string) *table.Table {
	return table.New().
		Headers(header...).
		Rows(rows...).
		Border(lipgloss.NormalBorder()).
		Wrap(false).
		StyleFunc(func(row, _ int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				s = s.Bold(true)
			}
			return s
		})
}

// renderList renders issue or pull-request items as a table.
func renderList(writer io.Writer, items []app.Item, emptyMessage string) {
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.Rkey, it.Title, it.State, it.Author.Handle, shortDate(it.UpdatedAt)})
	}
	renderTable(writer, []string{"RKEY", "TITLE", "STATE", "AUTHOR", "UPDATED"}, rows, emptyMessage)
}
