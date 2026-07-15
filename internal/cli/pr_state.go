package cli

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/knot"
	"github.com/spf13/cobra"
)

var (
	prStateRepo string
	prEditTitle string
	prEditBody  string
	prMergeRepo string
)

var prCloseCmd = newPRStateCmd("close", "closed")
var prReopenCmd = newPRStateCmd("reopen", "open")

var prEditCmd = &cobra.Command{
	Use:   "edit <rkey>",
	Short: "Edit a pull request",
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
		return editRecord(cmd.Context(), atClient, did, pullCollection, args[0], prEditTitle, prEditBody, setTitle, setBody)
	},
}

var prMergeCmd = &cobra.Command{
	Use:   "merge <rkey>",
	Short: "Merge a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}
		pullURI, repoURI, err := targetRecord(ctx, prMergeRepo, pullCollection, args[0])
		if err != nil {
			return err
		}
		knotHost, err := repoKnot(ctx, repoURI)
		if err != nil {
			return err
		}
		token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.merge")
		if err != nil {
			return err
		}
		if err := knot.New(knotHost, token).Merge(ctx, knot.MergeInput{Repo: repoURI, Pull: pullURI}); err != nil {
			return err
		}
		if err := putState(ctx, atClient, did, args[0], pullCollection, pullURI, "merged"); err != nil {
			return fmt.Errorf("record merged pull request status: %w", err)
		}
		return output(stateResult{Rkey: args[0], State: "merged"}, func(result stateResult) {
			fmt.Printf("Pull request %s merged\n", result.Rkey)
		})
	},
}

func newPRStateCmd(use, status string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <rkey>",
		Short: use + " a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			atClient, did, err := authenticatedATProto(cmd.Context())
			if err != nil {
				return err
			}
			target, _, err := targetRecord(cmd.Context(), prStateRepo, pullCollection, args[0])
			if err != nil {
				return err
			}
			if err := putState(cmd.Context(), atClient, did, args[0], pullCollection, target, status); err != nil {
				return fmt.Errorf("%s pull request: %w", use, err)
			}
			return output(stateResult{Rkey: args[0], State: status}, func(result stateResult) {
				fmt.Printf("Pull request %s %s\n", result.Rkey, result.State)
			})
		},
	}
}

func repoKnot(ctx context.Context, repoURI string) (string, error) {
	repo, err := client.GetRepo(ctx, repoURI)
	if err != nil {
		return "", fmt.Errorf("get repository: %w", err)
	}
	if repo.Value.Knot == "" {
		return "", fmt.Errorf("repository record has no knot")
	}
	return repo.Value.Knot, nil
}

func init() {
	prCloseCmd.Flags().StringVarP(&prStateRepo, "repo", "R", "", "Target repository as handle/repo")
	prReopenCmd.Flags().StringVarP(&prStateRepo, "repo", "R", "", "Target repository as handle/repo")
	prEditCmd.Flags().StringVarP(&prEditTitle, "title", "t", "", "New title")
	prEditCmd.Flags().StringVarP(&prEditBody, "body", "b", "", "New body")
	prMergeCmd.Flags().StringVarP(&prMergeRepo, "repo", "R", "", "Target repository as handle/repo")
}
