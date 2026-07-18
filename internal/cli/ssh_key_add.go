package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"
)

var sshKeyAddTitle string

var sshKeyAddCmd = &cobra.Command{
	Use:   "add [<key-file>]",
	Short: "Add an SSH key to your Tangled account",
	Long: `Add an SSH public key to your Tangled account.

If no key file is given, defaults to ~/.ssh/id_ed25519.pub.
Requires authentication (run "tg auth login" first).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		atClient, did, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}

		keyPath := "~/.ssh/id_ed25519.pub"
		if len(args) == 1 {
			keyPath = args[0]
		}
		if strings.HasPrefix(keyPath, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve home directory: %w", err)
			}
			keyPath = filepath.Join(home, keyPath[2:])
		}

		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("read key file %q: %w", keyPath, err)
		}
		key := strings.TrimSpace(string(keyBytes))
		if key == "" {
			return fmt.Errorf("key file %q is empty", keyPath)
		}

		title := sshKeyAddTitle
		if title == "" {
			title = filepath.Base(keyPath)
		}

		uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
			Repo:       did,
			Collection: "sh.tangled.publicKey",
			Rkey:       string(syntax.NewTIDNow(0)),
			Record: sshKeyRecord{
				Type:      "sh.tangled.publicKey",
				Key:       key,
				Name:      title,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		})
		if err != nil {
			return fmt.Errorf("add SSH key: %w", err)
		}

		result := sshKeyAddResult{Name: title, URI: uri}
		return output(result, func(added sshKeyAddResult) {
			fmt.Printf("Added SSH key %q (%s)\n", added.Name, added.URI)
		})
	},
}

func init() {
	sshKeyAddCmd.Flags().StringVarP(&sshKeyAddTitle, "title", "t", "", "Title for the new key")
}
