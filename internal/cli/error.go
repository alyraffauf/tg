package cli

import (
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
)

// renderError writes a concise error heading followed by the actionable error.
// Piped output stays free of ANSI escape sequences.
func renderError(writer io.Writer, err error) {
	if !isTerminal(writer) {
		fmt.Fprintf(writer, "Error: %s\n", err)
		return
	}

	heading := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Red).Render("✗ Error:")
	fmt.Fprintf(writer, "%s %s\n", heading, err)
}
