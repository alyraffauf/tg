package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

const (
	oauthCallbackAddr = "127.0.0.1:8095"
	oauthCallbackURL  = "http://" + oauthCallbackAddr + "/callback"
)

func NewRoot(service *app.Service) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "tg",
		Short:        "A CLI for Tangled",
		SilenceUsage: true,
	}
	configureRoot(rootCmd)

	auth := newAuthCommand(service)
	auth.AddCommand(newAuthLoginCommand(service), newAuthLogoutCommand(service), newAuthStatusCommand(service), newAuthTokenCommand(service), newAuthListCommand(service), newAuthSwitchCommand(service))
	rootCmd.AddCommand(auth)

	issue := newIssueCommand(service)
	issue.AddCommand(newIssueListCommand(service), newIssueViewCommand(service), newIssueCreateCommand(service), newIssueCommentCommand(service), newIssueCloseCommand(service), newIssueReopenCommand(service), newIssueEditCommand(service))
	rootCmd.AddCommand(issue)

	pull := newPRCommand(service)
	pull.AddCommand(newPRListCommand(service), newPRViewCommand(service), newPRCreateCommand(service), newPRCommentCommand(service), newPRDiffCommand(service), newPRCheckoutCommand(service), newPRCloseCommand(service), newPRReopenCommand(service), newPREditCommand(service), newPRMergeCommand(service))
	rootCmd.AddCommand(pull)

	repo := newRepoCommand(service)
	repo.AddCommand(newRepoViewCommand(service), newRepoCloneCommand(service), newRepoCreateCommand(service), newRepoListCommand(service), newRepoEditCommand(service), newRepoSetDefaultBranchCommand(service), newRepoDeleteCommand(service), newRepoForkCommand(service))
	rootCmd.AddCommand(repo)

	keys := newSSHKeyCommand(service)
	keys.AddCommand(newSSHKeyAddCommand(service), newSSHKeyListCommand(service), newSSHKeyDeleteCommand(service))
	rootCmd.AddCommand(keys)

	stringsCmd := newStringCommand(service)
	stringsCmd.AddCommand(newStringCreateCommand(service), newStringListCommand(service), newStringViewCommand(service), newStringDeleteCommand(service))
	rootCmd.AddCommand(stringsCmd, newBrowseCommand(service), newCompletionCommand(service), newManCommand(service), newAPICommand(service))
	return rootCmd
}

func Execute() error {
	return ExecuteWith(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}

// ExecuteWith runs the CLI with explicit arguments and I/O streams.
func ExecuteWith(arguments []string, input io.Reader, output, errorOutput io.Writer) error {
	flags, err := parseFlagSettings(arguments)
	if err != nil {
		return err
	}
	settings := loadConfig(flags, errorOutput)
	service := app.NewWithStreams(settings.Appview, oauthCallbackURL, output, errorOutput)
	service.SetAccount(settings.Account)
	root := NewRoot(service)
	root.SetArgs(arguments)
	root.SetIn(input)
	root.SetOut(output)
	root.SetErr(errorOutput)
	err = root.Execute()
	// A not-authenticated error from any service method is surfaced as the
	// familiar login hint, so individual commands don't each have to.
	if errors.Is(err, app.ErrNotAuthenticated) {
		return fmt.Errorf("not logged in; run \"tg auth login\" first")
	}
	return err
}

func parseFlagSettings(arguments []string) (flagSettings, error) {
	var flags flagSettings
	for index := 0; index < len(arguments); index++ {
		if arguments[index] == "--" {
			break
		}
		argument := arguments[index]
		name, value, hasValue := strings.Cut(argument, "=")
		switch name {
		case "--config":
			flags.ConfigPath = flagValue(arguments, index, value, hasValue)
			flags.ConfigSet = true
		case "--appview":
			flags.Appview = flagValue(arguments, index, value, hasValue)
			flags.AppviewSet = true
		case "--account":
			flags.Account = flagValue(arguments, index, value, hasValue)
			flags.AccountSet = true
		}
		if !hasValue && (name == "--config" || name == "--appview" || name == "--account") {
			if index+1 >= len(arguments) || arguments[index+1] == "--" {
				return flagSettings{}, fmt.Errorf("flag %s requires a value", name)
			}
			index++
		}
	}
	return flags, nil
}

func flagValue(arguments []string, index int, value string, hasValue bool) string {
	if hasValue {
		return value
	}
	if index+1 < len(arguments) {
		return arguments[index+1]
	}
	return ""
}

func configureRoot(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().String("config", "", "Path to config file (default: $XDG_CONFIG_HOME/tg/config.toml)")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	rootCmd.PersistentFlags().String("appview", defaultAppview, "Appview host URL (overrides config file and TG_APPVIEW)")
	rootCmd.PersistentFlags().String("account", "", "Account handle or DID to use (overrides the active account and TG_ACCOUNT)")
}
