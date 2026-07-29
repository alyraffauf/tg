package cli

import (
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthSwitchCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <handle-or-did>",
		Short: "Select the active account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := service.SwitchAccount(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return output(cmd, result, func(item *app.AuthAccountResult) {
				renderAuthSwitch(cmd.OutOrStdout(), item)
			})
		},
	}
}

func renderAuthSwitch(writer io.Writer, account *app.AuthAccountResult) {
	styles := newAuthOutputStyles(isTerminal(writer))
	writeAuthHeading(writer, styles.active, "✓", "Active account changed")
	writeAuthDetail(writer, styles, "Account", styles.value.Render(account.Handle))
	writeAuthDetail(writer, styles, "DID", styles.dim.Render(account.DID))
	writeAuthDetail(writer, styles, "Method", account.Method)
}
