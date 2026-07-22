package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRCommand(_ *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "pr",
		Short: "Manage pull requests on a Tangled repository",
	}
}
