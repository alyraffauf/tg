package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var (
	repoCreateDescription string
	repoCreateKnot        string
	repoCreateClone       bool
	repoCreatePushPath    string
	repoCreateRemote      string
)

var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a repository on Tangled",
	Long: `Create a repository on Tangled.

The repository is provisioned on a knot (default ` + knot.DefaultKnot + `) and a
sh.tangled.repo record is written to your PDS. The repository name is used as
the record key, matching the current Tangled schema.

Use --clone to clone the new repository into the current directory, or
--push=<path> to push an existing local repository at that path to the new
remote (and set its current branch as the default branch).

Requires authentication (run "tg auth login" first).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if auth == nil || !auth.IsAuthenticated() {
			return fmt.Errorf("not logged in; run \"tg auth login\" first")
		}

		pds, err := auth.APIClient(ctx)
		if err != nil {
			return fmt.Errorf("get auth client: %w", err)
		}
		atClient := &atproto.ATProto{Client: pds}
		did := auth.CurrentDID().String()

		knotHost := repoCreateKnot
		if knotHost == "" {
			knotHost = knot.DefaultKnot
		}

		uri, err := provisionRepo(ctx, atClient, provisionRepoInput{
			KnotHost:    knotHost,
			OwnerDID:    did,
			Name:        args[0],
			Description: repoCreateDescription,
		})
		if err != nil {
			return err
		}

		handle := ownerHandle(ctx, did)
		result := repoCreateResult{
			Handle: handle,
			Name:   args[0],
			URI:    uri,
			Knot:   knotHost,
		}

		if repoCreateClone {
			if err := gitutil.CloneRepo(ctx, gitutil.CloneRepoParams{
				Handle:  handle,
				Repo:    args[0],
				RepoDir: args[0],
			}); err != nil {
				return fmt.Errorf("clone new repository: %w", err)
			}
			result.Cloned = true
		}
		if repoCreatePushPath != "" {
			if err := pushToNewRepo(ctx, atClient, pushToNewRepoInput{
				KnotHost:   knotHost,
				RepoURI:    uri,
				Handle:     handle,
				RepoName:   args[0],
				PushPath:   repoCreatePushPath,
				RemoteName: repoCreateRemote,
			}); err != nil {
				return err
			}
			result.Pushed = true
		}

		return output(result, func(repo repoCreateResult) {
			fmt.Printf("Created repository %s/%s\n", repo.Handle, repo.Name)
			if repo.Cloned {
				fmt.Printf("Cloned into %s\n", repo.Name)
			}
			if repo.Pushed {
				fmt.Printf("Pushed to %s\n", repo.Name)
			}
		})
	},
}

func init() {
	repoCreateCmd.Flags().StringVar(&repoCreateDescription, "description", "", "Repository description")
	repoCreateCmd.Flags().StringVar(&repoCreateKnot, "knot", "", "knot host to create on (default "+knot.DefaultKnot+")")
	repoCreateCmd.Flags().BoolVar(&repoCreateClone, "clone", false, "Clone the new repository into the current directory")
	repoCreateCmd.Flags().StringVar(&repoCreatePushPath, "push", "", "Push an existing local repository at this path to the new remote (e.g. .)")
	repoCreateCmd.Flags().StringVar(&repoCreateRemote, "remote", "origin", "Remote name to use with --push")
}

type provisionRepoInput struct {
	KnotHost    string
	OwnerDID    string
	Name        string
	Description string
}

// provisionRepo creates the repo on the knot and writes the sh.tangled.repo
// record to the user's PDS.
func provisionRepo(ctx context.Context, atClient *atproto.ATProto, in provisionRepoInput) (string, error) {
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+in.KnotHost, "sh.tangled.repo.create")
	if err != nil {
		return "", err
	}
	repoDid, err := knot.New(in.KnotHost, token).CreateRepo(ctx, knot.CreateRepoInput{
		Name: in.Name,
		Rkey: in.Name,
	})
	if err != nil {
		return "", err
	}
	record := tangled.RepoRecord{
		Type:      "sh.tangled.repo",
		Knot:      in.KnotHost,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		RepoDid:   repoDid,
	}
	if in.Description != "" {
		record.Description = in.Description
	}
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       in.OwnerDID,
		Collection: "sh.tangled.repo",
		Rkey:       in.Name,
		Record:     record,
	})
	if err != nil {
		return "", err
	}
	return uri, nil
}

type pushToNewRepoInput struct {
	KnotHost   string
	RepoURI    string
	Handle     string
	RepoName   string
	PushPath   string
	RemoteName string
}

// pushToNewRepo sets the default branch then pushes. Default-branch failure is
// warned, not fatal. Set before push so the knot's hook skips its PR suggestion.
func pushToNewRepo(ctx context.Context, atClient *atproto.ATProto, in pushToNewRepoInput) error {
	branch, err := setDefaultBranch(ctx, atClient, setDefaultBranchInput{
		KnotHost: in.KnotHost,
		RepoURI:  in.RepoURI,
		Dir:      in.PushPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not set default branch: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Set default branch to %s\n", branch)
	}
	if err := gitutil.PushNewRepo(ctx, gitutil.PushNewRepoParams{
		Dir:        in.PushPath,
		Handle:     in.Handle,
		Repo:       in.RepoName,
		RemoteName: in.RemoteName,
	}); err != nil {
		return fmt.Errorf("push to new repository: %w", err)
	}
	return nil
}

type setDefaultBranchInput struct {
	KnotHost string
	RepoURI  string
	Dir      string
}

// setDefaultBranch repoints the default branch to the local repo's current
// branch. Mints a fresh token because the create token is lexicon-scoped.
func setDefaultBranch(ctx context.Context, atClient *atproto.ATProto, in setDefaultBranchInput) (string, error) {
	branch, err := gitutil.CurrentBranch(ctx, in.Dir)
	if err != nil {
		return "", err
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+in.KnotHost, "sh.tangled.repo.setDefaultBranch")
	if err != nil {
		return "", err
	}
	if err := knot.New(in.KnotHost, token).SetDefaultBranch(ctx, knot.SetDefaultBranchInput{
		Repo:          in.RepoURI,
		DefaultBranch: branch,
	}); err != nil {
		return branch, err
	}
	return branch, nil
}

func ownerHandle(ctx context.Context, did string) string {
	if ident, err := resolver.ResolveDID(ctx, did); err == nil {
		return ident.Handle.String()
	}
	return did
}
