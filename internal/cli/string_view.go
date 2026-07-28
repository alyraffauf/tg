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
				fields := []detailField{
					{"Filename", view.Filename},
					{"Author", view.Author.Handle},
					{"Created", view.CreatedAt},
				}
				if view.Description != "" {
					fields = append(fields, detailField{"Description", view.Description})
				}
				renderDetail(cmd.OutOrStdout(), fields, "")
				fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", view.Contents)
			})
		},
	}
}
