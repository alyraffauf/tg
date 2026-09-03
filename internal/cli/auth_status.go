package cli

import (
	"fmt"
	"io"

	"charm.land/lipgloss/v2"
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthStatusCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the active account",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.AuthStatus(cmd.Context())
			if err != nil {
				return err
			}
			return output(cmd, result, func(r *app.AuthStatusResult) {
				renderAuthStatus(cmd.OutOrStdout(), r)
			})
		},
	}
}

func renderAuthStatus(writer io.Writer, result *app.AuthStatusResult) {
	styles := newAuthOutputStyles(isTerminal(writer))
	if !result.Authenticated {
		writeAuthHeading(writer, styles.inactive, "○", "Not logged in")
		writeAuthDetail(writer, styles, "Action", "tg auth login <handle>")
		return
	}

	switch result.Status {
	case app.SessionStatusActive:
		writeAuthHeading(writer, styles.active, "✓", "Authenticated")
		writeAuthIdentity(writer, styles, result)
		writeAuthDetail(writer, styles, "Session", "Active")
	case app.SessionStatusExpired:
		writeAuthHeading(writer, styles.warning, "!", "Session expired")
		writeAuthIdentity(writer, styles, result)
		writeAuthDetail(writer, styles, "Action", "tg auth login "+result.Handle)
	case app.SessionStatusUnknown:
		writeAuthHeading(writer, styles.warning, "?", "Could not verify session")
		writeAuthIdentity(writer, styles, result)
		writeAuthDetail(writer, styles, "Reason", "Network error")
		writeAuthDetail(writer, styles, "Retry", "tg auth status")
	}
}

type authOutputStyles struct {
	active   lipgloss.Style
	inactive lipgloss.Style
	warning  lipgloss.Style
	label    lipgloss.Style
	value    lipgloss.Style
	dim      lipgloss.Style
}

func newAuthOutputStyles(colorEnabled bool) authOutputStyles {
	styles := authOutputStyles{}
	if !colorEnabled {
		return styles
	}
	styles.active = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Green)
	styles.inactive = lipgloss.NewStyle().Faint(true)
	styles.warning = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow)
	styles.label = lipgloss.NewStyle().Faint(true)
	styles.value = lipgloss.NewStyle().Bold(true)
	styles.dim = lipgloss.NewStyle().Faint(true)
	return styles
}

func writeAuthHeading(writer io.Writer, style lipgloss.Style, icon, text string) {
	fmt.Fprintf(writer, "%s %s\n", style.Render(icon), style.Render(text))
}

func writeAuthIdentity(writer io.Writer, styles authOutputStyles, result *app.AuthStatusResult) {
	if result.Handle != "" {
		writeAuthDetail(writer, styles, "Account", styles.value.Render(result.Handle))
	}
	if result.DID != "" {
		writeAuthDetail(writer, styles, "DID", styles.dim.Render(result.DID))
	}
}

func writeAuthDetail(writer io.Writer, styles authOutputStyles, label, value string) {
	fmt.Fprintf(writer, "  %s  %s\n", styles.label.Render(label), value)
}
