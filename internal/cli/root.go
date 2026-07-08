package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/spf13/cobra"
)

const (
	oauthCallbackAddr = "127.0.0.1:8095"
	oauthCallbackURL  = "http://" + oauthCallbackAddr + "/callback"
)

var (
	resolver = &atproto.Resolver{Directory: identity.DefaultDirectory()}
	client   = &tangled.Tangled{
		Client: &xrpc.Client{Host: "https://api.tangled.org"},
		Logger: slog.Default(),
	}
	auth *atproto.AuthManager
)

var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "A CLI for Tangled",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	initAuth()

	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)

	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)

	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prCheckoutCmd)

	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoCloneCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoListCmd)
}

func initAuth() {
	dir, err := atproto.ConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not determine config dir: %v\n", err)
		return
	}

	auth, err = atproto.NewAuthManager(oauthCallbackURL, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: auth initialization failed: %v\n", err)
	}
}
