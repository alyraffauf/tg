package cli

import "github.com/spf13/cobra"

// sshKeyRecord is the value of a sh.tangled.publicKey record.
type sshKeyRecord struct {
	Type      string `json:"$type"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

var sshKeyCmd = &cobra.Command{
	Use:   "ssh-key",
	Short: "Manage SSH keys on Tangled",
}
