package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/knot"
	"github.com/spf13/cobra"
)

var repoDeleteConfirm bool

var repoDeleteCmd = &cobra.Command{
	Use:   "delete [handle/repo]",
	Short: "Delete a Tangled repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !repoDeleteConfirm {
			return fmt.Errorf("refusing to delete without --yes")
		}
		ctx := cmd.Context()
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}
		handle, name, err := resolveTarget(ctx, args)
		if err != nil {
			return err
		}
		repo, err := requireOwnedRepo(ctx, handle, name, did)
		if err != nil {
			return err
		}
		if repo.Value.Knot == "" {
			return fmt.Errorf("repo %q has no knot", handle+"/"+name)
		}
		rkey := extractRKey(repo.URI)
		existingRecord, getErr := atClient.GetRecord(ctx, did, "sh.tangled.repo", rkey)
		// getErr is non-fatal: the record may already be deleted. Only
		// call DeleteRecord if it still exists.

		token, err := atClient.GetServiceAuth(ctx, "did:web:"+repo.Value.Knot, "sh.tangled.repo.delete")
		if err != nil {
			return fmt.Errorf("get knot authorization: %w", err)
		}
		if getErr == nil {
			if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
				Repo:       did,
				Collection: "sh.tangled.repo",
				Rkey:       rkey,
			}); err != nil {
				return fmt.Errorf("delete repository record: %w", err)
			}
		}
		if err := knot.New(repo.Value.Knot, token).DeleteRepo(ctx, knot.DeleteRepoInput{
			DID:  did,
			Name: name,
			Rkey: rkey,
		}); err != nil {
			if getErr == nil {
				if _, _, restoreErr := atClient.PutRecord(ctx, atproto.PutRecordInput{
					Repo: did, Collection: "sh.tangled.repo", Rkey: rkey, Record: existingRecord.Value,
				}); restoreErr != nil {
					return fmt.Errorf("delete knot repository: %w; restore repository record: %v", err, restoreErr)
				}
			}
			return err
		}
		return output(repoDeleteResult{URI: repo.URI}, func(result repoDeleteResult) {
			fmt.Printf("Deleted repository %s\n", result.URI)
		})
	},
}

func init() {
	repoDeleteCmd.Flags().BoolVar(&repoDeleteConfirm, "yes", false, "Confirm permanent repository deletion")
}

type repoDeleteResult struct {
	URI string `json:"uri"`
}
