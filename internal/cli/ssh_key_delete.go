package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var sshKeyDeleteCmd = &cobra.Command{
	Use:   "delete <rkey>",
	Short: "Delete an SSH key from your Tangled account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}
		if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
			Repo:       did,
			Collection: "sh.tangled.publicKey",
			Rkey:       args[0],
		}); err != nil {
			return fmt.Errorf("delete SSH key: %w", err)
		}
		return output(deletedRecordResult{Rkey: args[0]}, func(result deletedRecordResult) {
			fmt.Printf("Deleted SSH key %s\n", result.Rkey)
		})
	},
}

type deletedRecordResult struct {
	Rkey string `json:"rkey"`
}
