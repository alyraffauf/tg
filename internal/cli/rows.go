package cli

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/alyraffauf/tg/internal/app"
	xterm "github.com/charmbracelet/x/term"
)

// shortDate renders an ISO 8601 timestamp as the date in the user's timezone.
func shortDate(timestamp string) string {
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return parsed.Local().Format("2006-01-02")
}

// localTimestamp renders an ISO 8601 timestamp in the user's timezone.
func localTimestamp(timestamp string) string {
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return timestamp
	}
	return parsed.Local().Format("2006-01-02 15:04:05 MST")
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
// is a terminal the table uses a muted header and separator; when piped or
// redirected it falls back to a plain tab-aligned table.
// emptyMessage is shown when rows is empty.
func renderTable(writer io.Writer, header []string, rows [][]string, emptyMessage string) {
	if len(rows) == 0 {
		fmt.Fprintln(writer, emptyMessage)
		return
	}

	if isTerminal(writer) {
		renderTerminalTable(writer, header, rows)
		return
	}

	tw := tabwriter.NewWriter(writer, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
}

// renderTerminalTable renders a styled table for interactive terminals. Cells
// truncate (with "…") rather than wrap; on narrow terminals the table is
// shrunk to fit the terminal width.
func renderTerminalTable(writer io.Writer, header []string, rows [][]string) {
	rendered := newTerminalTable(header, rows).Render()

	// The top border spans the full table width, so it tells us whether
	// the natural width fits the terminal. If it overflows, re-render
	// with a width cap so lipgloss shrinks columns instead of spilling.
	if w := terminalWidth(writer); w > 0 {
		if first, _, _ := strings.Cut(rendered, "\n"); lipgloss.Width(first) > w {
			rendered = newTerminalTable(header, rows).Width(w).Render()
		}
	}

	fmt.Fprintln(writer, rendered)
}

func newTerminalTable(header []string, rows [][]string) *table.Table {
	return table.New().
		Headers(header...).
		Rows(rows...).
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(true).
		BorderStyle(lipgloss.NewStyle().Faint(true)).
		Wrap(false).
		StyleFunc(func(row, column int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return s.Bold(true).Faint(true)
			}
			return styleTableCell(s, header, rows, row, column)
		})
}

func styleTableCell(style lipgloss.Style, header []string, rows [][]string, row, column int) lipgloss.Style {
	if row < 0 || row >= len(rows) || column < 0 || column >= len(header) || column >= len(rows[row]) {
		return style
	}

	columnName := strings.ToLower(header[column])
	value := strings.ToLower(rows[row][column])
	switch columnName {
	case "active":
		if rows[row][column] == "✓" {
			return style.Bold(true).Foreground(lipgloss.Green)
		}
	case "state":
		if terminalColor := stateColor(value); terminalColor != nil {
			return style.Foreground(terminalColor)
		}
	case "status":
		if terminalColor := workflowStatusColor(value); terminalColor != nil {
			return style.Foreground(terminalColor)
		}
	case "title", "account":
		return style.Bold(true)
	case "did", "id", "rkey", "method", "updated":
		return style.Faint(true)
	}
	return style
}

func workflowStatusColor(status string) color.Color {
	status = strings.TrimLeft(strings.ToLower(status), "✓✗●○⊘ ")
	switch status {
	case "success":
		return lipgloss.Green
	case "failed", "timeout":
		return lipgloss.Red
	case "running":
		return lipgloss.Blue
	case "pending":
		return lipgloss.Yellow
	default:
		return nil
	}
}

// renderList renders issue or pull-request items as a table.
func renderList(writer io.Writer, items []app.Item, emptyMessage string) {
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.Rkey, it.Title, it.State, it.Author.Handle, shortDate(it.UpdatedAt)})
	}
	renderTable(writer, []string{"RKEY", "TITLE", "STATE", "AUTHOR", "UPDATED"}, rows, emptyMessage)
}
