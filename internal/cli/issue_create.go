package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newIssueCreateCommand(service *app.Service) *cobra.Command {
	var bodyText, bodyFile, repository string

	command := &cobra.Command{
		Use:   "create <title>",
		Short: "Create an issue on a Tangled repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			body, err := commandBody(bodyText, bodyFile)
			if err != nil {
				return err
			}
			target, err := resolveTargetFlag(ctx, repository, service)
			if err != nil {
				return err
			}
			result, err := service.CreateIssue(ctx, target, args[0], body)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.CreatedRecordResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Created issue %s\n", result.URI)
			})
		},
	}
	command.Flags().StringVarP(&bodyText, "body", "b", "", "Issue body")
	command.Flags().StringVarP(&bodyFile, "body-file", "F", "", "Read issue body from file")
	command.Flags().StringVarP(&repository, "repo", "R", "", "Target repository as handle/repo")
	return command
}
