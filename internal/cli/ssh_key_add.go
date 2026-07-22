package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newSSHKeyAddCommand(service *app.Service) *cobra.Command {
	var title string

	command := &cobra.Command{
		Use:   "add [<key-file>]",
		Short: "Add an SSH key to your Tangled account",
		Long: `Add an SSH public key to your Tangled account.

If no key file is given, defaults to ~/.ssh/id_ed25519.pub.
Requires authentication (run "tg auth login" first).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

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

			keyTitle := title
			if keyTitle == "" {
				keyTitle = filepath.Base(keyPath)
			}

			result, err := service.AddSSHKey(ctx, keyTitle, key)
			if err != nil {
				return err
			}
			return output(cmd, result, func(added *app.SSHKeyAddResult) {
				fmt.Fprintf(cmd.OutOrStdout(), "Added SSH key %q (%s)\n", added.Name, added.URI)
			})
		},
	}
	command.Flags().StringVarP(&title, "title", "t", "", "Title for the new key")
	return command
}
