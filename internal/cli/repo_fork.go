package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoForkCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "fork <handle/repo> [name]",
		Short: "Fork a Tangled repository",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			source, err := app.ParseTarget(args[0])
			if err != nil {
				return err
			}
			name := source.Repo
			if len(args) == 2 {
				name = args[1]
			}
			result, err := service.ForkRepo(ctx, source, name)
			if err != nil {
				return err
			}
			return output(cmd, result, func(fork *app.RepoForkResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Forked %s as %s/%s\n", source, fork.Handle, fork.Name)
			})
		},
	}
}
