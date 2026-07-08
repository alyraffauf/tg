package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/atclient"
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

		ident, err := resolver.ResolveHandle(ctx, handle)
		if err != nil {
			return fmt.Errorf("resolve handle %q: %w", handle, err)
		}

		pdsURL, err := resolver.ResolvePDS(ctx, ident.DID.String())
		if err != nil {
			return fmt.Errorf("resolve PDS for %q: %w", handle, err)
		}

		atClient := &atproto.ATProto{Client: &atclient.APIClient{Host: pdsURL}}
		out, err := atClient.ListRecords(ctx, ident.DID.String(), "sh.tangled.publicKey", atproto.ListRecordsOpts{Limit: defaultListLimit})
		if err != nil {
			return fmt.Errorf("list SSH keys for %q: %w", handle, err)
		}

		items := buildSSHKeyItems(out.Records)
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
	if len(items) == 0 {
		fmt.Println("No SSH keys found.")
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, "NAME\tKEY\tADDED")

	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", item.Name, item.Key, shortDate(item.CreatedAt))
	}
	tw.Flush()
}
