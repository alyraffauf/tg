package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var stringDeleteCmd = &cobra.Command{
	Use:   "delete <rkey>",
	Short: "Delete a string from your Tangled account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}
		if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
			Repo:       did,
			Collection: stringCollection,
			Rkey:       args[0],
		}); err != nil {
			return fmt.Errorf("delete string: %w", err)
		}
		return output(deletedRecordResult{Rkey: args[0]}, func(result deletedRecordResult) {
			fmt.Printf("Deleted string %s\n", result.Rkey)
		})
	},
}
