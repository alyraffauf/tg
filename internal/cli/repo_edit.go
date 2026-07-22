package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoEditCommand(service *app.Service) *cobra.Command {
	var description, website, spindle string
	var addLabels, removeLabels []string

	command := &cobra.Command{
		Use:   "edit [handle/repo]",
		Short: "Edit a Tangled repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("description") && !cmd.Flags().Changed("website") && !cmd.Flags().Changed("spindle") && len(addLabels) == 0 && len(removeLabels) == 0 {
				return fmt.Errorf("set a repository field to update")
			}
			ctx := cmd.Context()
			target, err := resolveTarget(ctx, args, service)
			if err != nil {
				return err
			}
			in := app.EditRepoInput{AddLabels: addLabels, RemoveLabels: removeLabels}
			if cmd.Flags().Changed("description") {
				in.Description = &description
			}
			if cmd.Flags().Changed("website") {
				in.Website = &website
			}
			if cmd.Flags().Changed("spindle") {
				in.Spindle = &spindle
			}
			result, err := service.EditRepo(ctx, target, in)
			if err != nil {
				return err
			}
			return output(cmd, result, func(result *app.RepoEditResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated repository %s\n", result.URI)
			})
		},
	}
	command.Flags().StringVarP(&description, "description", "d", "", "Repository description")
	command.Flags().StringVar(&website, "website", "", "Repository website")
	command.Flags().StringVar(&spindle, "spindle", "", "Repository spindle")
	command.Flags().StringSliceVar(&addLabels, "add-label", nil, "Label to add")
	command.Flags().StringSliceVar(&removeLabels, "remove-label", nil, "Label to remove")
	return command
}
