package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
)

// ViewRepo fetches a single repository record.
func (s *Service) ViewRepo(ctx context.Context, t Target) (*RepoItem, error) {
	ident, err := s.Resolver.ResolveHandle(ctx, t.Handle)
	if err != nil {
		return nil, fmt.Errorf("resolve handle %q: %w", t.Handle, err)
	}
	repoURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, t.Repo)
	tangledRepo, err := s.Appview.GetRepo(ctx, repoURI)
	if err != nil {
		return nil, fmt.Errorf("get repo %s: %w", t, err)
	}
	name := tangledRepo.Value.Name
	if name == "" {
		name = t.Repo
	}
	return &RepoItem{
		Name:        name,
		Author:      t.Handle,
		URI:         repoURI,
		Knot:        tangledRepo.Value.Knot,
		Description: tangledRepo.Value.Description,
		CreatedAt:   tangledRepo.Value.CreatedAt,
		RepoDid:     tangledRepo.Value.RepoDid,
	}, nil
}

// ListRepos lists every repository owned by handle.
func (s *Service) ListRepos(ctx context.Context, handle string) ([]RepoItem, error) {
	ident, err := s.Resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return nil, fmt.Errorf("resolve handle %q: %w", handle, err)
	}
	repos, err := s.Appview.ListRepos(ctx, ident.DID.String())
	if err != nil {
		return nil, fmt.Errorf("list repos for %q: %w", handle, err)
	}
	return buildRepoItems(repos.Items, handle), nil
}

func buildRepoItems(items []tangled.Repo, author string) []RepoItem {
	result := make([]RepoItem, 0, len(items))
	for _, tangledRepo := range items {
		name := tangledRepo.Value.Name
		if name == "" {
			// Fall back to the rkey segment of the at:// URI.
			if idx := strings.LastIndex(tangledRepo.URI, "/"); idx != -1 {
				name = tangledRepo.URI[idx+1:]
			}
		}
		result = append(result, RepoItem{
			Name:        name,
			URI:         tangledRepo.URI,
			Author:      author,
			Knot:        tangledRepo.Value.Knot,
			Description: tangledRepo.Value.Description,
			CreatedAt:   tangledRepo.Value.CreatedAt,
			RepoDid:     tangledRepo.Value.RepoDid,
		})
	}
	return result
}

// ProvisionRepoInput configures repository provisioning.
type ProvisionRepoInput struct {
	KnotHost    string
	Name        string
	Description string
}

// ProvisionRepo creates the repo on the knot and writes the sh.tangled.repo
// record to the user's PDS. Returns the new record URI and the owner's handle.
func (s *Service) ProvisionRepo(ctx context.Context, in ProvisionRepoInput) (uri, handle string, err error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return "", "", err
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+in.KnotHost, "sh.tangled.repo.create")
	if err != nil {
		return "", "", err
	}
	repoDid, err := knot.New(in.KnotHost, token).CreateRepo(ctx, knot.CreateRepoInput{
		Name: in.Name,
		Rkey: in.Name,
	})
	if err != nil {
		return "", "", err
	}
	record := tangled.RepoRecord{
		Type:      repoCollection,
		Knot:      in.KnotHost,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		RepoDid:   repoDid,
	}
	if in.Description != "" {
		record.Description = in.Description
	}
	uri, _, err = atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: repoCollection,
		Rkey:       in.Name,
		Record:     record,
	})
	if err != nil {
		return "", "", err
	}
	return uri, s.OwnerHandle(ctx, did), nil
}

// OwnerHandle resolves did to a handle, falling back to the DID string.
func (s *Service) OwnerHandle(ctx context.Context, did string) string {
	if ident, err := s.Resolver.ResolveDID(ctx, did); err == nil {
		return ident.Handle.String()
	}
	return did
}

// SetRepoDefaultBranch sets the default branch of the authenticated user's
// repo t.
func (s *Service) SetRepoDefaultBranch(ctx context.Context, t Target, branch string) (*RepoDefaultBranchResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.RequireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	if repo.Value.Knot == "" {
		return nil, fmt.Errorf("repo %q has no knot", t.String())
	}
	if err := s.setKnotDefaultBranch(ctx, atClient, repo.Value.Knot, repo.URI, branch); err != nil {
		return nil, err
	}
	return &RepoDefaultBranchResult{URI: repo.URI, Branch: branch}, nil
}

// SetDefaultBranchFromDir repoints the default branch of repoURI on knotHost
// to the current branch of the local git repository at dir. It is best-effort
// during repo creation; callers may warn rather than fail on error.
func (s *Service) SetDefaultBranchFromDir(ctx context.Context, knotHost, repoURI, dir string) (string, error) {
	atClient, _, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return "", err
	}
	branch, err := s.Git.CurrentBranch(ctx, dir)
	if err != nil {
		return "", err
	}
	if err := s.setKnotDefaultBranch(ctx, atClient, knotHost, repoURI, branch); err != nil {
		return branch, err
	}
	return branch, nil
}

// PushNewRepo sets the knot default branch when possible, then pushes the
// local repository to its new Tangled remote. A default-branch error is
// returned separately because repository creation treats it as a warning.
func (s *Service) PushNewRepo(ctx context.Context, in PushNewRepoInput) (string, error, error) {
	branch, defaultBranchErr := s.SetDefaultBranchFromDir(ctx, in.KnotHost, in.RepoURI, in.Dir)
	if err := s.Git.PushNewRepo(ctx, gitutil.PushNewRepoParams{
		Dir: in.Dir, Handle: in.Handle, Repo: in.Repo, RemoteName: in.RemoteName,
	}); err != nil {
		return branch, defaultBranchErr, fmt.Errorf("push to new repository: %w", err)
	}
	return branch, defaultBranchErr, nil
}

// PushNewRepoInput configures pushing a newly created repository.
type PushNewRepoInput struct {
	KnotHost   string
	RepoURI    string
	Dir        string
	Handle     string
	Repo       string
	RemoteName string
}

func (s *Service) setKnotDefaultBranch(ctx context.Context, atClient *atproto.ATProto, knotHost, repoURI, branch string) error {
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.setDefaultBranch")
	if err != nil {
		return fmt.Errorf("get knot authorization: %w", err)
	}
	return knot.New(knotHost, token).SetDefaultBranch(ctx, knot.SetDefaultBranchInput{
		Repo:          repoURI,
		DefaultBranch: branch,
	})
}

// EditRepoInput configures repository edits. Pointer fields are nil when the
// corresponding flag was not set.
type EditRepoInput struct {
	Description  *string
	Website      *string
	Spindle      *string
	AddLabels    []string
	RemoveLabels []string
}

// EditRepo patches repository fields on the authenticated user's repo t.
func (s *Service) EditRepo(ctx context.Context, t Target, in EditRepoInput) (*RepoEditResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.RequireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	rkey := extractRKey(repo.URI)
	existing, err := atClient.GetRecord(ctx, did, repoCollection, rkey)
	if err != nil {
		return nil, fmt.Errorf("get repository record: %w", err)
	}
	record, err := repoRecordMap(existing.Value)
	if err != nil {
		return nil, err
	}
	if in.Description != nil {
		record["description"] = *in.Description
	}
	if in.Website != nil {
		record["website"] = *in.Website
	}
	if in.Spindle != nil {
		record["spindle"] = *in.Spindle
	}
	if len(in.AddLabels) > 0 || len(in.RemoveLabels) > 0 {
		labels := labelsFromRecord(record["labels"])
		for _, label := range in.AddLabels {
			labels[label] = true
		}
		for _, label := range in.RemoveLabels {
			delete(labels, label)
		}
		record["labels"] = labelNames(labels)
	}
	if _, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: repoCollection,
		Rkey:       rkey,
		Record:     record,
	}); err != nil {
		return nil, fmt.Errorf("edit repository: %w", err)
	}
	result := &RepoEditResult{URI: repo.URI}
	if in.Description != nil {
		result.Description = *in.Description
	}
	return result, nil
}

func repoRecordMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode repository record: %w", err)
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("decode repository record: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("repository record is not an object")
	}
	return record, nil
}

func labelsFromRecord(value any) map[string]bool {
	labels := make(map[string]bool)
	values, ok := value.([]any)
	if !ok {
		return labels
	}
	for _, value := range values {
		if label, ok := value.(string); ok {
			labels[label] = true
		}
	}
	return labels
}

func labelNames(labels map[string]bool) []string {
	names := make([]string, 0, len(labels))
	for label := range labels {
		names = append(names, label)
	}
	sort.Strings(names)
	return names
}

// DeleteRepo deletes the repository record and the knot-side repo. If the
// knot deletion fails after the record is deleted, the record is restored.
func (s *Service) DeleteRepo(ctx context.Context, t Target) (*RepoDeleteResult, error) {
	atClient, did, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.RequireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	if repo.Value.Knot == "" {
		return nil, fmt.Errorf("repo %q has no knot", t.String())
	}
	rkey := extractRKey(repo.URI)
	existingRecord, getErr := atClient.GetRecord(ctx, did, repoCollection, rkey)
	// getErr is non-fatal: the record may already be deleted. Only call
	// DeleteRecord if it still exists.

	token, err := atClient.GetServiceAuth(ctx, "did:web:"+repo.Value.Knot, "sh.tangled.repo.delete")
	if err != nil {
		return nil, fmt.Errorf("get knot authorization: %w", err)
	}
	if getErr == nil {
		if err := atClient.DeleteRecord(ctx, atproto.DeleteRecordInput{
			Repo:       did,
			Collection: repoCollection,
			Rkey:       rkey,
		}); err != nil {
			return nil, fmt.Errorf("delete repository record: %w", err)
		}
	}
	if err := knot.New(repo.Value.Knot, token).DeleteRepo(ctx, knot.DeleteRepoInput{
		DID:  did,
		Name: t.Repo,
		Rkey: rkey,
	}); err != nil {
		if getErr == nil {
			if _, _, restoreErr := atClient.PutRecord(ctx, atproto.PutRecordInput{
				Repo: did, Collection: repoCollection, Rkey: rkey, Record: existingRecord.Value,
			}); restoreErr != nil {
				return nil, fmt.Errorf("delete knot repository: %w; restore repository record: %v", err, restoreErr)
			}
		}
		return nil, err
	}
	return &RepoDeleteResult{URI: repo.URI}, nil
}

// ForkRepo creates a fork of source on the authenticated user's account,
// named name (defaults to the source repo's name).
func (s *Service) ForkRepo(ctx context.Context, source Target, name string) (*RepoForkResult, error) {
	atClient, ownerDID, err := s.AuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}

	src, err := s.getForkSource(ctx, source)
	if err != nil {
		return nil, err
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+src.Knot, "sh.tangled.repo.create")
	if err != nil {
		return nil, fmt.Errorf("get knot service auth: %w", err)
	}
	repoDID, err := knot.New(src.Knot, token).CreateRepo(ctx, knot.CreateRepoInput{
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
		Record: tangled.RepoRecord{
			Type:      repoCollection,
			Name:      name,
			Knot:      src.Knot,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			RepoDid:   repoDID,
			Source:    src.URI,
		},
	})
	if err != nil {
		cleanupErr := s.deleteFork(ctx, atClient, src.Knot, ownerDID, name)
		if cleanupErr != nil {
			return nil, fmt.Errorf("write fork record: %w; delete orphaned fork: %v", err, cleanupErr)
		}
		return nil, fmt.Errorf("write fork record: %w", err)
	}
	return &RepoForkResult{Handle: s.OwnerHandle(ctx, ownerDID), Name: name, URI: uri, Knot: src.Knot}, nil
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
	ident, err := s.Resolver.ResolveHandle(ctx, t.Handle)
	if err != nil {
		return forkSource{}, fmt.Errorf("resolve handle %q: %w", t.Handle, err)
	}
	uri := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, t.Repo)
	repo, err := s.Appview.GetRepo(ctx, uri)
	if err != nil {
		return forkSource{}, fmt.Errorf("get source repository %s: %w", t, err)
	}
	if repo.Value.Knot == "" {
		return forkSource{}, fmt.Errorf("source repository %s has no knot", t)
	}
	if repo.Value.RepoDid == "" {
		return forkSource{}, fmt.Errorf("source repository %s has no repo DID", t)
	}
	if repo.URI != "" {
		uri = repo.URI
	}
	return forkSource{URI: uri, Knot: repo.Value.Knot, RepoDID: repo.Value.RepoDid}, nil
}

func (s *Service) deleteFork(ctx context.Context, atClient *atproto.ATProto, knotHost, did, name string) error {
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.delete")
	if err != nil {
		return fmt.Errorf("get knot authorization: %w", err)
	}
	if err := knot.New(knotHost, token).DeleteRepo(ctx, knot.DeleteRepoInput{DID: did, Name: name, Rkey: name}); err != nil {
		return err
	}
	return nil
}
