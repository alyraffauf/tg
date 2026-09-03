package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPipelineTriggerCommand(service *app.Service) *cobra.Command {
	var repository string
	var workflows []string

	command := &cobra.Command{
		Use:   "trigger <revision>",
		Short: "Trigger pipelines for a Git revision",
		Long: `Trigger pipelines for a commit or branch.

A full commit SHA works outside a Git checkout. tg resolves other revisions in
the current checkout. Without --repo, tg uses the origin remote.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			result, err := service.TriggerPipeline(cmd.Context(), target, args[0], workflows)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.PipelineTriggerResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Triggered pipeline %s.\n", result.Pipeline)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringSliceVarP(&workflows, "workflow", "w", nil, "Workflow to trigger (repeatable)")
	return command
}
