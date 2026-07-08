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

		renderSSHKeys(out.Records)
		return nil
	},
}

func renderSSHKeys(items []atproto.RecordItem) {
	if len(items) == 0 {
		fmt.Println("No SSH keys found.")
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, "NAME\tKEY\tADDED")

	for _, item := range items {
		var rec sshKeyRecord
		data, err := json.Marshal(item.Value)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", rec.Name, rec.Key, shortDate(rec.CreatedAt))
	}
	tw.Flush()
}
