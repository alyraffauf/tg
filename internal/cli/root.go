package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/spf13/cobra"
)

const (
	oauthCallbackAddr = "127.0.0.1:8095"
	oauthCallbackURL  = "http://" + oauthCallbackAddr + "/callback"
)

var (
	resolver = &atproto.Resolver{Directory: identity.DefaultDirectory()}
	client   = &tangled.Tangled{
		Client: &atclient.APIClient{Host: "https://bobbin.klbr.net"},
		Logger: slog.Default(),
	}
	auth *atproto.AuthManager

	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "A CLI for Tangled",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	initAuth()

	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)

	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)

	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prCheckoutCmd)
	prCmd.AddCommand(prViewCmd)

	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoViewCmd)
	repoCmd.AddCommand(repoCloneCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoListCmd)

	rootCmd.AddCommand(sshKeyCmd)
	sshKeyCmd.AddCommand(sshKeyAddCmd)
	sshKeyCmd.AddCommand(sshKeyListCmd)
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
