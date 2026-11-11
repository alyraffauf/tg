package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newPRMarkMergedCommand(service *app.Service) *cobra.Command {
	var repository, expectedCID string
	command := &cobra.Command{
		Use:   "mark-merged <record-key>",
		Short: "Record an existing manual merge",
		Long:  "Record a pull request as manually merged. The code must already be integrated. This only updates the pull request status.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolveTargetFlag(cmd.Context(), repository, service)
			if err != nil {
				return err
			}
			result, err := service.MarkPullMerged(cmd.Context(), app.MarkPullMergedInput{Target: target, Rkey: args[0], ExpectedCID: expectedCID})
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.StateResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Pull request %s recorded as merged\n", result.Rkey)
			})
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	command.Flags().StringVar(&expectedCID, "expected-cid", "", "Reject changes unless this record CID is current")
	return command
}
