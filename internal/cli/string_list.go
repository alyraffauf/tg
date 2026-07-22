package cli

import (
	"io"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newStringListCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list [handle]",
		Short: "List strings on a Tangled account",
		Long: `List strings on a Tangled account.

If no argument is given, lists the authenticated user's strings
(run "tg auth login" first).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			handle, err := resolveHandleOrSelf(ctx, args, service)
			if err != nil {
				return err
			}
			items, err := service.ListStrings(ctx, handle)
			if err != nil {
				return err
			}
			return output(cmd, items, func(items []app.StringItem) {
				renderStringList(cmd.OutOrStdout(), items)
			})
		},
	}
}

func renderStringList(writer io.Writer, items []app.StringItem) {
	rows := make([][]string, 0, len(items))
	for _, str := range items {
		rows = append(rows, []string{str.Rkey, str.Filename, str.Description, shortDate(str.CreatedAt)})
	}
	renderTable(writer, []string{"RKEY", "FILENAME", "DESCRIPTION", "CREATED"}, rows, "No strings found.")
}
