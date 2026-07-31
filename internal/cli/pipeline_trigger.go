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
		Short: "Trigger pipelines for a commit or branch",
		Long: `Trigger pipelines for a commit or branch.

A full commit SHA works without a local checkout. Other Git revisions, such
as HEAD or a branch name, are resolved from the current checkout. If --repo
is not set, the repository is detected from the current directory's git
origin remote.`,
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
	command.Flags().StringSliceVarP(&workflows, "workflow", "w", nil, "Workflow name to trigger (repeatable)")
	return command
}
