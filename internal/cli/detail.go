package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// defaultTerminalWidth is used when the terminal width can't be detected.
const defaultTerminalWidth = 80

type detailField struct {
	label string
	value string
}

// renderDetail writes aligned label/value pairs plus an optional markdown
// body. A writer-aware lipgloss renderer keeps TTY and piped output in one
// code path; the body renders as markdown on a terminal, verbatim otherwise.
func renderDetail(writer io.Writer, fields []detailField, body string) {
	renderer := lipgloss.NewRenderer(writer)
	labelStyle := renderer.NewStyle().Bold(true).Width(labelColumnWidth(fields))

	for _, field := range fields {
		fmt.Fprintf(writer, "%s%s\n", labelStyle.Render(field.label+":"), field.value)
	}

	if body == "" {
		return
	}

	fmt.Fprintln(writer)
	renderMarkdown(writer, body)
}

// labelColumnWidth returns the width to pad every label to; the +2 accounts
// for the trailing colon and one separator space.
func labelColumnWidth(fields []detailField) int {
	maxLabelLength := 0
	for _, field := range fields {
		if len(field.label) > maxLabelLength {
			maxLabelLength = len(field.label)
		}
	}
	return maxLabelLength + 2
}

// renderMarkdown renders body as styled markdown on a terminal, wrapping to
// the terminal width. When piped (or if rendering fails) it writes the raw
// body verbatim.
func renderMarkdown(writer io.Writer, body string) {
	if !isTerminal(writer) {
		fmt.Fprintln(writer, body)
		return
	}

	width := terminalWidth(writer)
	if width == 0 {
		width = defaultTerminalWidth
	}

	renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(width))
	if err != nil {
		fmt.Fprintln(writer, body)
		return
	}

	rendered, err := renderer.Render(body)
	if err != nil {
		fmt.Fprintln(writer, body)
		return
	}

	fmt.Fprintln(writer, strings.TrimSpace(rendered))
}
