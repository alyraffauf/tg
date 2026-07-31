package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoViewCommand(service *app.Service) *cobra.Command {
	command := &cobra.Command{
		Use:   "view <handle/repo>",
		Short: "View a Tangled repository",
		Long:  `View details for a Tangled repository, including the latest default-branch pipeline status when available.`,
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
			if status, err := service.PipelineStatus(ctx, target); err == nil {
				item.Pipeline = status.Pipeline
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: pipeline status unavailable: %v\n", err)
			}
			return output(cmd, item, func(item *app.RepoItem) {
				fields := []detailField{
					{"Name", item.Name},
					{"Description", item.Description},
					{"URI", item.URI},
					{"Knot", item.Knot},
					{"Created", item.CreatedAt},
				}
				if item.RepoDid != "" {
					fields = append(fields, detailField{"Repo DID", item.RepoDid})
				}
				if item.Pipeline != nil {
					fields = append(fields, detailField{"Pipeline", pipelineStatusSummary(cmd.OutOrStdout(), item.Pipeline.Workflows)})
				}
				renderDetail(cmd.OutOrStdout(), fields, "")
			})
		},
	}
	return command
}
