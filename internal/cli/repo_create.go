package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newRepoCreateCommand(createRepo func(context.Context, app.CreateRepoInput) (*app.RepoCreateResult, error), defaultKnot, defaultSSHPort, defaultProtocol string) *cobra.Command {
	var description, knotHost, pushPath, remote, sshPort, policy string
	var clone bool

	command := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a repository",
		Long: `Create a repository on Tangled.

By default, tg uses ` + app.DefaultKnot + `. Use --knot to choose a Knot.

Use --clone to clone the new repository. Use --push=<path> to push an existing
local repository and set its current branch as the default branch.

Run "tg auth login" first.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			parsedSSHPort, err := parseRepoCreateSSHPort(sshPort, knotHost, defaultProtocol, clone, pushPath)
			if err != nil {
				return err
			}
			result, err := createRepo(ctx, app.CreateRepoInput{
				ContributionPolicy: policy, KnotHost: knotHost, SSHPort: parsedSSHPort, Name: args[0], Description: description,
				Clone: clone, CloneProtocol: defaultProtocol, PushPath: pushPath, RemoteName: remote,
			})
			if result != nil {
				if outputErr := output(cmd, result, func(repo *app.RepoCreateResult) { renderRepoCreate(cmd, repo) }); outputErr != nil {
					return outputErr
				}
			}
			return err
		},
	}
	command.Flags().StringVar(&policy, "contribution-policy", "", "Who may contribute: anyone, members or collaborators (default: knot policy)")
	command.Flags().StringVar(&description, "description", "", "Repository description")
	command.Flags().StringVar(&knotHost, "knot", defaultKnot, "Knot host")
	command.Flags().StringVar(&sshPort, "ssh-port", defaultSSHPort, "SSH port for an explicitly selected Knot")
	command.Flags().BoolVar(&clone, "clone", false, "Clone the new repository")
	command.Flags().StringVar(&pushPath, "push", "", "Push the repository at this path, such as .")
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
	if !repo.Declared && repo.RepoDID != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Repository %s has no confirmed owner declaration; inspect it before retrying\n", repo.RepoDID)
	}
	if repo.Cloned {
		fmt.Fprintf(cmd.OutOrStdout(), "Cloned into %s\n", repo.Name)
	}
	if repo.Pushed {
		fmt.Fprintf(cmd.OutOrStdout(), "Pushed to %s\n", repo.Name)
	}
	if repo.DefaultBranch != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "Set default branch to %s\n", repo.DefaultBranch)
	}
}
