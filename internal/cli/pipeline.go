package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPipelineCommand(_ *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "pipeline",
		Short: "Inspect and manage CI pipelines",
	}
}
