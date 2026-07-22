package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newStringCommand(_ *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "string",
		Short: "Manage strings on Tangled",
	}
}
