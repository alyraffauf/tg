package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newStringViewCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "view <rkey> [handle]",
		Short: "View a string on a Tangled account",
		Long: `View a string by its rkey (the last segment of its at:// URI).

If no handle is given, views the authenticated user's string
(run "tg auth login" first).`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			rkey := args[0]

			handle, err := resolveHandleOrSelf(ctx, args[1:], service)
			if err != nil {
				return err
			}
			result, err := service.ViewString(ctx, handle, rkey)
			if err != nil {
				return err
			}
			return output(cmd, result, func(view *app.StringViewResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Filename:    %s\n", view.Filename)
				fmt.Fprintf(cmd.OutOrStdout(), "Author:      %s\n", view.Author.Handle)
				fmt.Fprintf(cmd.OutOrStdout(), "Created:     %s\n", view.CreatedAt)
				if view.Description != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", view.Description)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", view.Contents)
			})
		},
	}
}
