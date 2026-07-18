package cli

import (
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/atproto"
	"github.com/spf13/cobra"
)

var stringListCmd = &cobra.Command{
	Use:   "list [handle]",
	Short: "List strings on a Tangled account",
	Long: `List strings on a Tangled account.

If no argument is given, lists the authenticated user's strings
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

		records, err := atClient.ListAllRecords(ctx, did, stringCollection, atproto.ListRecordsOpts{Limit: defaultListLimit})
		if err != nil {
			return fmt.Errorf("list strings for %q: %w", handle, err)
		}

		items := buildStringItems(records)
		return output(items, renderStringList)
	},
}

func buildStringItems(records []atproto.RecordItem) []stringItem {
	items := make([]stringItem, 0, len(records))
	for _, rec := range records {
		var str stringRecord
		data, err := json.Marshal(rec.Value)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &str); err != nil {
			continue
		}
		// Records without a filename are not strings; skip them rather
		// than rendering a blank row.
		if str.Filename == "" {
			continue
		}
		items = append(items, stringItem{
			Rkey:        extractRKey(rec.URI),
			URI:         rec.URI,
			Filename:    str.Filename,
			Description: str.Description,
			CreatedAt:   str.CreatedAt,
		})
	}
	return items
}

func renderStringList(items []stringItem) {
	rows := make([][]string, 0, len(items))
	for _, str := range items {
		rows = append(rows, []string{str.Rkey, str.Filename, str.Description, shortDate(str.CreatedAt)})
	}
	renderTable([]string{"RKEY", "FILENAME", "DESCRIPTION", "CREATED"}, rows, "No strings found.")
}
