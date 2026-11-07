package cli

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

type repoForkService interface {
	ForkRepoOnKnot(ctx context.Context, source app.Target, name, knotHost string) (*app.RepoForkResult, error)
}

func newRepoForkCommand(service repoForkService) *cobra.Command {
	var knotHost string
	command := &cobra.Command{
		Use:   "fork <handle/repo> [name]",
		Short: "Fork a repository",
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
			result, err := service.ForkRepoOnKnot(ctx, source, name, knotHost)
			// A failed fork can leave a repository behind; report it before the error.
			if result != nil {
				if outputErr := output(cmd, result, func(fork *app.RepoForkResult) { renderRepoFork(cmd, source, fork) }); outputErr != nil {
					return outputErr
				}
			}
			return err
		},
	}
	command.Flags().StringVar(&knotHost, "knot", "", "Knot host (default: source repository's Knot)")
	return command
}

func renderRepoFork(cmd *cobra.Command, source app.Target, fork *app.RepoForkResult) {
	fmt.Fprintf(cmd.OutOrStdout(), "Forked %s as %s/%s\n", source, fork.Handle, fork.Name)
	if !fork.Declared && fork.RepoDID != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Repository %s has no confirmed owner declaration; inspect it before retrying\n", fork.RepoDID)
	}
	if fork.ImportStatus != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Import: %s\n", fork.ImportStatus)
	}
}
