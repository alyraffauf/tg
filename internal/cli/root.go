package cli

import (
	"log/slog"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/spf13/cobra"
)

var (
	resolver = &atproto.Resolver{Directory: identity.DefaultDirectory()}
	client   = &tangled.Tangled{
		Client: &xrpc.Client{Host: "https://api.tangled.org"},
		Logger: slog.Default(),
	}
)

var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "A CLI for Tangled",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)

	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prCheckoutCmd)

	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoCloneCmd)
}
