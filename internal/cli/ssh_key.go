package cli

import (
	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newSSHKeyCommand(_ *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "ssh-key",
		Short: "Manage SSH keys on Tangled",
	}
}
