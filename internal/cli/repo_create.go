package cli

import (
	"fmt"
	"strconv"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoCreateCommand(service *app.Service, defaultKnot, defaultSSHPort, defaultProtocol string) *cobra.Command {
	var description, knotHost, pushPath, remote, sshPort string
	var clone bool

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a repository on Tangled",
		Long: `Create a repository on Tangled.

The repository is provisioned on the Knot selected by --knot, TG_KNOT, or tg
configuration. When none is selected, tg reads your sh.tangled.knot
registrations from your PDS and verifies each Knot's sh.tangled.owner response
against your DID. At most 10 registrations are considered; select a Knot
explicitly when you have more. One verified Knot is selected automatically.
If no registrations exist, tg falls back to ` + app.DefaultKnot + `. Registrations
with no successful verification stop creation. Failed registrations produce
warnings when another registration succeeds, and multiple verified Knots
require an explicit selection. Discovered hosts are contacted through your
machine's normal DNS and HTTPS networking; private hosts are allowed. A
sh.tangled.repo record is written to your PDS. The repository name is used as
the record key, matching the current Tangled schema.

Use --clone to clone the new repository into the current directory, or
--push=<path> to push an existing local repository at that path to the new
remote (and set its current branch as the default branch).

Requires authentication (run "tg auth login" first).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			parsedSSHPort, err := parseRepoCreateSSHPort(sshPort, knotHost, defaultProtocol, clone, pushPath)
			if err != nil {
				return err
			}
			result, err := service.CreateRepo(ctx, app.CreateRepoInput{
				KnotHost: knotHost, SSHPort: parsedSSHPort, Name: args[0], Description: description,
				Clone: clone, CloneProtocol: defaultProtocol, PushPath: pushPath, RemoteName: remote,
			})
			if err != nil {
				return err
			}
			return output(cmd, result, func(repo *app.RepoCreateResult) { renderRepoCreate(cmd, repo) })
		},
	}
	command.Flags().StringVar(&description, "description", "", "Repository description")
	command.Flags().StringVar(&knotHost, "knot", defaultKnot, "Knot host to provision and optionally push to (overrides TG_KNOT, config, and automatic discovery)")
	command.Flags().StringVar(&sshPort, "ssh-port", defaultSSHPort, "SSH port for cloning from or pushing to an explicitly selected Knot (overrides config file and TG_SSH_PORT)")
	command.Flags().BoolVar(&clone, "clone", false, "Clone the new repository into the current directory")
	command.Flags().StringVar(&pushPath, "push", "", "Push an existing local repository at this path to the new remote (e.g. .)")
	command.Flags().StringVar(&remote, "remote", "origin", "Remote name to use with --push")
	return command
}

func parseRepoCreateSSHPort(sshPort, knotHost, cloneProtocol string, clone bool, pushPath string) (int, error) {
	usesDirectSSH := knotHost != "" && (pushPath != "" || clone && cloneProtocol == "ssh")
	if !usesDirectSSH {
		return 0, nil
	}
	parsedSSHPort, err := strconv.Atoi(sshPort)
	if err != nil {
		return 0, fmt.Errorf("invalid SSH port %q: %w", sshPort, err)
	}
	return parsedSSHPort, nil
}

func renderRepoCreate(cmd *cobra.Command, repo *app.RepoCreateResult) {
	fmt.Fprintf(cmd.OutOrStdout(), "Created repository %s/%s\n", repo.Handle, repo.Name)
	if repo.Cloned {
		fmt.Fprintf(cmd.OutOrStdout(), "Cloned into %s\n", repo.Name)
	}
	if repo.Pushed {
		fmt.Fprintf(cmd.OutOrStdout(), "Pushed to %s\n", repo.Name)
	}
	if repo.DefaultBranch != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Set default branch to %s\n", repo.DefaultBranch)
	}
	for _, warning := range repo.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", warning)
	}
}
