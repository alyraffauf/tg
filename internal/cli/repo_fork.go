package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var repoForkCmd = &cobra.Command{
	Use:   "fork <handle/repo> [name]",
	Short: "Fork a Tangled repository",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		atClient, ownerDID, err := authenticatedATProto(ctx)
		if err != nil {
			return err
		}

		handle, sourceName, err := parseHandleRepo(args[0])
		if err != nil {
			return err
		}
		name := sourceName
		if len(args) == 2 {
			name = args[1]
		}

		source, err := getForkSource(ctx, handle, sourceName)
		if err != nil {
			return err
		}
		token, err := atClient.GetServiceAuth(ctx, "did:web:"+source.Knot, "sh.tangled.repo.create")
		if err != nil {
			return fmt.Errorf("get knot service auth: %w", err)
		}
		repoDID, err := knot.New(source.Knot, token).CreateRepo(ctx, knot.CreateRepoInput{
			Name:   name,
			Rkey:   name,
			Source: forkSourceURL(source.Knot, source.RepoDID),
		})
		if err != nil {
			return err
		}
		uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
			Repo:       ownerDID,
			Collection: "sh.tangled.repo",
			Rkey:       name,
			Record: tangled.RepoRecord{
				Type:      "sh.tangled.repo",
				Name:      name,
				Knot:      source.Knot,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
				RepoDid:   repoDID,
				Source:    source.URI,
			},
		})
		if err != nil {
			cleanupErr := deleteFork(ctx, atClient, source.Knot, ownerDID, name)
			if cleanupErr != nil {
				return fmt.Errorf("write fork record: %w; delete orphaned fork: %v", err, cleanupErr)
			}
			return fmt.Errorf("write fork record: %w", err)
		}

		return output(repoForkResult{Handle: ownerHandle(ctx, ownerDID), Name: name, URI: uri, Knot: source.Knot}, func(fork repoForkResult) {
			fmt.Printf("Forked %s/%s as %s/%s\n", handle, sourceName, fork.Handle, fork.Name)
		})
	},
}

func deleteFork(ctx context.Context, atClient *atproto.ATProto, knotHost, did, name string) error {
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.delete")
	if err != nil {
		return fmt.Errorf("get knot authorization: %w", err)
	}
	if err := knot.New(knotHost, token).DeleteRepo(ctx, knot.DeleteRepoInput{DID: did, Name: name, Rkey: name}); err != nil {
		return err
	}
	return nil
}

type forkSource struct {
	URI     string
	Knot    string
	RepoDID string
}

func forkSourceURL(knotHost, repoDID string) string {
	base := strings.TrimRight(knotHost, "/")
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	return base + "/" + repoDID
}

func getForkSource(ctx context.Context, handle, name string) (forkSource, error) {
	ident, err := resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return forkSource{}, fmt.Errorf("resolve handle %q: %w", handle, err)
	}
	uri := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, name)
	repo, err := client.GetRepo(ctx, uri)
	if err != nil {
		return forkSource{}, fmt.Errorf("get source repository %s/%s: %w", handle, name, err)
	}
	if repo.Value.Knot == "" {
		return forkSource{}, fmt.Errorf("source repository %s/%s has no knot", handle, name)
	}
	if repo.Value.RepoDid == "" {
		return forkSource{}, fmt.Errorf("source repository %s/%s has no repo DID", handle, name)
	}
	if repo.URI != "" {
		uri = repo.URI
	}
	return forkSource{URI: uri, Knot: repo.Value.Knot, RepoDID: repo.Value.RepoDid}, nil
}

type repoForkResult struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	URI    string `json:"uri"`
	Knot   string `json:"knot"`
}
