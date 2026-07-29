package cli

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
)

// defaultTerminalWidth is used when the terminal width can't be detected.
const defaultTerminalWidth = 80

type detailField struct {
	label string
	value string
}

// renderDetail writes aligned label/value pairs plus an optional markdown
// body. The body renders as markdown on a terminal, verbatim otherwise.
func renderDetail(writer io.Writer, fields []detailField, body string) {
	labelStyle := lipgloss.NewStyle().Width(labelColumnWidth(fields))
	if isTerminal(writer) {
		labelStyle = labelStyle.Faint(true)
	}

	for _, field := range fields {
		fmt.Fprintf(writer, "%s%s\n", labelStyle.Render(field.label+":"), field.value)
	}

	if body == "" {
		return
	}

	fmt.Fprintln(writer)
	renderMarkdown(writer, body)
}

func formatDetailState(writer io.Writer, state string) string {
	if !isTerminal(writer) {
		return state
	}

	style := lipgloss.NewStyle()
	if terminalColor := stateColor(state); terminalColor != nil {
		style = style.Foreground(terminalColor)
	} else {
		style = style.Faint(true)
	}
	return style.Render(state)
}

func stateColor(state string) color.Color {
	switch strings.ToLower(state) {
	case "open":
		return lipgloss.Green
	case "closed":
		return lipgloss.Red
	case "merged":
		return lipgloss.Magenta
	case "draft":
		return lipgloss.Yellow
	default:
		return nil
	}
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

	// Pick the style from the terminal background. Query the same terminal that
	// receives the rendered output; if it isn't file-backed or the query fails,
	// default to dark.
	style := "dark"
	terminal := asFile(writer)
	if terminal != nil && !lipgloss.HasDarkBackground(os.Stdin, terminal) {
		style = "light"
	}

	renderer, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width))
	if err != nil {
		fmt.Fprintln(writer, body)
		return
	}

	rendered, err := renderer.Render(body)
	if err != nil {
		fmt.Fprintln(writer, body)
		return
	}

	// glamour v2 emits TrueColor, lipgloss downsamples to the terminal's color
	// profile on write.
	lipgloss.Fprintln(writer, strings.TrimSpace(rendered))
}
