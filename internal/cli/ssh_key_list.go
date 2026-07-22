package cli

import (
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newSSHKeyListCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list [handle]",
		Short: "List SSH keys on a Tangled account",
		Long: `List SSH keys on a Tangled account.

If no argument is given, lists the authenticated user's keys
(run "tg auth login" first).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			handle, err := resolveHandleOrSelf(ctx, args, service)
			if err != nil {
				return err
			}
			items, err := service.ListSSHKeys(ctx, handle)
			if err != nil {
				return err
			}
			return output(cmd, items, func(items []app.SSHKeyItem) {
				renderSSHKeyList(cmd.OutOrStdout(), items)
			})
		},
	}
}

func renderSSHKeyList(writer io.Writer, items []app.SSHKeyItem) {
	rows := make([][]string, 0, len(items))
	for _, key := range items {
		rows = append(rows, []string{key.Name, key.Key, shortDate(key.CreatedAt)})
	}
	renderTable(writer, []string{"NAME", "KEY", "ADDED"}, rows, "No SSH keys found.")
}
