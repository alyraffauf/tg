package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Service) ForkRepo(ctx context.Context, source Target, name string) (*RepoForkResult, error) {
	return s.ForkRepoOnKnot(ctx, source, name, "")
}

// ForkRepoOnKnot creates a fork on knotHost. When knotHost is empty, it uses
// the source repository's Knot for backwards-compatible same-Knot forks.
func (s *Service) ForkRepoOnKnot(ctx context.Context, source Target, name, knotHost string) (*RepoForkResult, error) {
	if _, err := syntax.ParseRecordKey(name); err != nil {
		return nil, fmt.Errorf("invalid repository name %q: %w", name, err)
	}
	atClient, ownerDID, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}

	src, err := s.getForkSource(ctx, source)
	if err != nil {
		return nil, err
	}
	selectedKnot := src.Knot
	if knotHost != "" {
		selectedKnot, err = parseKnotHostname(knotHost)
		if err != nil {
			return nil, err
		}
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+selectedKnot, "sh.tangled.repo.create")
	if err != nil {
		return nil, fmt.Errorf("get knot service auth: %w", err)
	}
	repoDID, err := s.knot.New(selectedKnot, token).CreateRepo(ctx, knot.CreateRepoInput{
		Name:   name,
		Rkey:   name,
		Source: forkSourceURL(src.Knot, src.RepoDID),
	})
	if err != nil {
		return nil, err
	}
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       ownerDID,
		Collection: repoCollection,
		Rkey:       name,
		Record: tangledlex.Repo{
			LexiconTypeID: repoCollection,
			Name:          optionalString(name),
			Knot:          selectedKnot,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			RepoDid:       optionalString(repoDID),
			Source:        optionalString(forkSourceURL(src.Knot, src.RepoDID)),
		},
	})
	if err != nil {
		cleanupErr := s.deleteFork(ctx, atClient, selectedKnot, ownerDID, name)
		if cleanupErr != nil {
			return nil, fmt.Errorf("write fork record: %w; delete orphaned fork: %v", err, cleanupErr)
		}
		return nil, fmt.Errorf("write fork record: %w", err)
	}
	return &RepoForkResult{Handle: s.ownerHandle(ctx, ownerDID), Name: name, URI: uri, Knot: selectedKnot}, nil
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

func (s *Service) getForkSource(ctx context.Context, t Target) (forkSource, error) {
	// Repository names are not guaranteed to be their ATProto record rkeys.
	// Use the common resolver so forks work for repositories with generated rkeys.
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return forkSource{}, fmt.Errorf("resolve source repository %s: %w", t, err)
	}
	if repo.Value.Knot == "" {
		return forkSource{}, fmt.Errorf("source repository %s has no knot", t)
	}
	if stringValue(repo.Value.RepoDid) == "" {
		return forkSource{}, fmt.Errorf("source repository %s has no repo DID", t)
	}
	return forkSource{URI: repo.URI, Knot: repo.Value.Knot, RepoDID: stringValue(repo.Value.RepoDid)}, nil
}

func (s *Service) deleteFork(ctx context.Context, atClient pdsClient, knotHost, did, name string) error {
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.delete")
	if err != nil {
		return fmt.Errorf("get knot authorization: %w", err)
	}
	if err := s.knot.New(knotHost, token).DeleteRepo(ctx, knot.DeleteRepoInput{DID: did, Name: name, Rkey: name}); err != nil {
		return err
	}
	return nil
}
