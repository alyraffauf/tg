package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	issueStateRepo string
	issueEditTitle string
	issueEditBody  string
)

var issueCloseCmd = newIssueStateCmd("close", "closed")
var issueReopenCmd = newIssueStateCmd("reopen", "open")

var issueEditCmd = &cobra.Command{
	Use:   "edit <rkey>",
	Short: "Edit an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		setTitle := cmd.Flags().Changed("title")
		setBody := cmd.Flags().Changed("body")
		if !setTitle && !setBody {
			return fmt.Errorf("set --title or --body")
		}
		atClient, did, err := authenticatedATProto(cmd.Context())
		if err != nil {
			return err
		}
		return editRecord(cmd.Context(), atClient, did, issueCollection, args[0], issueEditTitle, issueEditBody, setTitle, setBody)
	},
}

func newIssueStateCmd(use, state string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <rkey>",
		Short: use + " an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			atClient, did, err := authenticatedATProto(cmd.Context())
			if err != nil {
				return err
			}
			target, _, err := targetRecord(cmd.Context(), issueStateRepo, issueCollection, args[0])
			if err != nil {
				return err
			}
			if err := putState(cmd.Context(), atClient, did, args[0], issueCollection, target, state); err != nil {
				return fmt.Errorf("%s issue: %w", use, err)
			}
			return output(stateResult{Rkey: args[0], State: state}, func(result stateResult) {
				fmt.Printf("Issue %s %s\n", result.Rkey, result.State)
			})
		},
	}
}

func init() {
	issueCloseCmd.Flags().StringVarP(&issueStateRepo, "repo", "R", "", "Target repository as handle/repo")
	issueReopenCmd.Flags().StringVarP(&issueStateRepo, "repo", "R", "", "Target repository as handle/repo")
	issueEditCmd.Flags().StringVarP(&issueEditTitle, "title", "t", "", "New title")
	issueEditCmd.Flags().StringVarP(&issueEditBody, "body", "b", "", "New body")
}
