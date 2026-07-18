package cli

import (
	"log/slog"

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
		Client: &atclient.APIClient{Host: defaultAppview},
		Logger: slog.Default(),
	}
	auth = atproto.NewAuthManager(oauthCallbackURL)

	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "A CLI for Tangled",
	// Errors such as "not logged in" are expected and shouldn't dump usage.
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		client.Client.Host = config.GetString("appview")
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default: $XDG_CONFIG_HOME/tg/config.toml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().String("appview", defaultAppview, "Appview host URL (overrides config file and TG_APPVIEW)")

	config.BindPFlag("appview", rootCmd.PersistentFlags().Lookup("appview"))

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

	rootCmd.AddCommand(stringCmd)
	stringCmd.AddCommand(stringCreateCmd)
	stringCmd.AddCommand(stringListCmd)
	stringCmd.AddCommand(stringViewCmd)
	stringCmd.AddCommand(stringDeleteCmd)

	rootCmd.AddCommand(browseCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(manCmd)
	rootCmd.AddCommand(apiCmd)
}
