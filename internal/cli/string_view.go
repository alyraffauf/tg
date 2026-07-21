package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var stringViewCmd = &cobra.Command{
	Use:   "view <rkey> [handle]",
	Short: "View a string on a Tangled account",
	Long: `View a string by its rkey (the last segment of its at:// URI).

If no handle is given, views the authenticated user's string
(run "tg auth login" first).`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		rkey := args[0]

		handle, err := resolveHandleOrSelf(ctx, args[1:])
		if err != nil {
			return err
		}

		atClient, did, err := publicAccountReader(ctx, handle)
		if err != nil {
			return err
		}

		found, err := atClient.GetRecord(ctx, did, stringCollection, rkey)
		if err != nil {
			return fmt.Errorf("get string %q for %q: %w", rkey, handle, err)
		}

		record, err := decodeStringRecord(found.Value)
		if err != nil {
			return fmt.Errorf("decode string %q: %w", rkey, err)
		}

		result := stringViewResult{
			Rkey:        rkey,
			URI:         found.URI,
			Filename:    record.Filename,
			Author:      author{DID: did, Handle: handle},
			Description: record.Description,
			Contents:    record.Contents,
			CreatedAt:   record.CreatedAt,
		}
		return output(result, func(view stringViewResult) {
			fmt.Printf("Filename:    %s\n", view.Filename)
			fmt.Printf("Author:      %s\n", view.Author.Handle)
			fmt.Printf("Created:     %s\n", view.CreatedAt)
			if view.Description != "" {
				fmt.Printf("Description: %s\n", view.Description)
			}
			fmt.Printf("\n%s\n", view.Contents)
		})
	},
}

// decodeStringRecord decodes a record value into a stringRecord.
func decodeStringRecord(value any) (stringRecord, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return stringRecord{}, fmt.Errorf("encode record: %w", err)
	}
	var record stringRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return stringRecord{}, fmt.Errorf("decode record: %w", err)
	}
	return record, nil
}
