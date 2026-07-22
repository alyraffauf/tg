package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoCommand(_ *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories on Tangled",
	}
}
