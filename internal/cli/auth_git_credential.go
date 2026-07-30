package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAuthGitCredentialCommand(service *app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "git-credential <operation>",
		Short: "Git credential helper for HTTPS push to Tangled knots",
		Long: `Provide HTTPS credentials for Git pushes to a Tangled knot.

Configure the helper for a knot:

    git config --global credential."https://<knot>".helper "!tg auth git-credential"

Point the remote at that knot:

    git remote set-url origin https://<knot>/<handle>/<repo>.git

The helper mints a short-lived token only for the current repository's recorded
knot. It does not store credentials.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "get":
				return gitCredentialGet(cmd, service)
			case "store", "erase":
				return nil
			default:
				return fmt.Errorf("unknown git-credential operation %q (want get, store, or erase)", args[0])
			}
		},
	}
}

func gitCredentialGet(cmd *cobra.Command, service *app.Service) error {
	credentialAttributes, err := readCredentialInput(cmd.InOrStdin())
	if err != nil {
		return err
	}
	if !isHTTPSCredentialRequest(credentialAttributes) {
		return nil
	}
	credentials, err := service.GitPushToken(cmd.Context(), credentialAttributes["host"])
	if err != nil {
		return err
	}
	if !credentials.MatchesRequestedHost {
		return nil
	}
	out := cmd.OutOrStdout()
	if credentials.Handle != "" {
		fmt.Fprintf(out, "username=%s\n", credentials.Handle)
	}
	fmt.Fprintf(out, "password=%s\n", credentials.Token)
	fmt.Fprintln(out)
	return nil
}

// readCredentialInput parses git's key=value credential attributes from stdin,
// stopping at a blank line or EOF.
func readCredentialInput(input io.Reader) (map[string]string, error) {
	credentialAttributes := make(map[string]string)
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			credentialAttributes[key] = value
		}
	}
	return credentialAttributes, scanner.Err()
}

func isHTTPSCredentialRequest(credentialAttributes map[string]string) bool {
	return strings.EqualFold(credentialAttributes["protocol"], "https") && strings.TrimSpace(credentialAttributes["host"]) != ""
}
