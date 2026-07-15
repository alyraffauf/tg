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
		Client: &atclient.APIClient{Host: appviewHost()},
		Logger: slog.Default(),
	}
	auth *atproto.AuthManager

	jsonOutput bool
	appview    string
)

func appviewHost() string {
	if host := os.Getenv("TG_APPVIEW"); host != "" {
		return host
	}
	return "https://bobbin.klbr.net"
}

var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "A CLI for Tangled",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		client.Client.Host = appview
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().StringVar(&appview, "appview", appviewHost(), "Appview host URL (overrides the TG_APPVIEW environment variable)")

	initAuth()

	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authTokenCmd)

	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueCommentCmd)
	issueCmd.AddCommand(issueCloseCmd)
	issueCmd.AddCommand(issueReopenCmd)
	issueCmd.AddCommand(issueEditCmd)

	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prViewCmd)
	prCmd.AddCommand(prCreateCmd)
	prCmd.AddCommand(prCommentCmd)
	prCmd.AddCommand(prDiffCmd)
	prCmd.AddCommand(prCheckoutCmd)
	prCmd.AddCommand(prCloseCmd)
	prCmd.AddCommand(prReopenCmd)
	prCmd.AddCommand(prEditCmd)
	prCmd.AddCommand(prMergeCmd)

	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoViewCmd)
	repoCmd.AddCommand(repoCloneCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoEditCmd)
	repoCmd.AddCommand(repoSetDefaultBranchCmd)
	repoCmd.AddCommand(repoDeleteCmd)
	repoCmd.AddCommand(repoForkCmd)

	rootCmd.AddCommand(sshKeyCmd)
	sshKeyCmd.AddCommand(sshKeyAddCmd)
	sshKeyCmd.AddCommand(sshKeyListCmd)
	sshKeyCmd.AddCommand(sshKeyDeleteCmd)

	rootCmd.AddCommand(browseCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(apiCmd)
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
