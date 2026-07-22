package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthCommand(_ *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
}
