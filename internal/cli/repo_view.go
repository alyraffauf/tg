package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoViewCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "view <handle/repo>",
		Short: "View a Tangled repository",
		Long:  `View details for a Tangled repository.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			target, err := app.ParseTarget(args[0])
			if err != nil {
				return err
			}
			item, err := service.ViewRepo(ctx, target)
			if err != nil {
				return err
			}
			return output(cmd, item, func(item *app.RepoItem) {
				fmt.Fprintf(cmd.OutOrStdout(), "Name:        %s\n", item.Name)
				fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", item.Description)
				fmt.Fprintf(cmd.OutOrStdout(), "URI:         %s\n", item.URI)
				fmt.Fprintf(cmd.OutOrStdout(), "Knot:        %s\n", item.Knot)
				fmt.Fprintf(cmd.OutOrStdout(), "Created:     %s\n", item.CreatedAt)
				if item.RepoDid != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Repo DID:    %s\n", item.RepoDid)
				}
			})
		},
	}
}
