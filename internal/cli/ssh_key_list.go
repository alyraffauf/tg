package cli

import (
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var sshKeyListCmd = &cobra.Command{
	Use:   "list [handle]",
	Short: "List SSH keys on a Tangled account",
	Long: `List SSH keys on a Tangled account.

If no argument is given, lists the authenticated user's keys
(run "tg auth login" first).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		handle, err := resolveHandleOrSelf(ctx, args)
		if err != nil {
			return err
		}

		atClient, did, err := publicAccountReader(ctx, handle)
		if err != nil {
			return err
		}

		records, err := atClient.ListAllRecords(ctx, did, "sh.tangled.publicKey", atproto.ListRecordsOpts{Limit: defaultListLimit})
		if err != nil {
			return fmt.Errorf("list SSH keys for %q: %w", handle, err)
		}

		items := buildSSHKeyItems(records)
		return output(items, renderSSHKeyList)
	},
}

func buildSSHKeyItems(records []atproto.RecordItem) []sshKeyItem {
	items := make([]sshKeyItem, 0, len(records))
	for _, rec := range records {
		var key sshKeyRecord
		data, err := json.Marshal(rec.Value)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &key); err != nil {
			continue
		}
		items = append(items, sshKeyItem{
			Name:      key.Name,
			Key:       key.Key,
			CreatedAt: key.CreatedAt,
			URI:       rec.URI,
		})
	}
	return items
}

func renderSSHKeyList(items []sshKeyItem) {
	rows := make([][]string, 0, len(items))
	for _, key := range items {
		rows = append(rows, []string{key.Name, key.Key, shortDate(key.CreatedAt)})
	}
	renderTable([]string{"NAME", "KEY", "ADDED"}, rows, "No SSH keys found.")
}
