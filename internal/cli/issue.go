package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newIssueCommand(_ *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "issue",
		Short: "Manage issues on a Tangled repository",
	}
}
