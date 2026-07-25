package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const (
	knotCollection       = "sh.tangled.knot"
	maxKnotRegistrations = 10
)

type knotRegistration struct {
	LexiconTypeID string `json:"$type"`
	CreatedAt     string `json:"createdAt"`
}

// ViewRepo fetches a single repository record.
func (s *Service) ViewRepo(ctx context.Context, t Target) (*RepoItem, error) {
	tangledRepo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	name := stringValue(tangledRepo.Value.Name)
	if name == "" {
		name = t.Repo
	}
	return &RepoItem{
		Name:        name,
		Author:      t.Handle,
		URI:         tangledRepo.URI,
		Knot:        tangledRepo.Value.Knot,
		Description: stringValue(tangledRepo.Value.Description),
		CreatedAt:   tangledRepo.Value.CreatedAt,
		RepoDid:     stringValue(tangledRepo.Value.RepoDid),
	}, nil
}

// ListRepos lists every repository owned by handle.
func (s *Service) ListRepos(ctx context.Context, handle string) ([]RepoItem, error) {
	ident, err := s.resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return nil, fmt.Errorf("resolve handle %q: %w", handle, err)
	}
	repos, err := s.appview.ListRepos(ctx, ident.DID.String())
	if err != nil {
		return nil, fmt.Errorf("list repos for %q: %w", handle, err)
	}
	return buildRepoItems(repos.Items, handle), nil
}

func buildRepoItems(items []tangled.Repo, author string) []RepoItem {
	canonicalRepos := canonicalRepoItems(items)
	result := make([]RepoItem, 0, len(canonicalRepos))
	for _, tangledRepo := range canonicalRepos {
		name := stringValue(tangledRepo.Value.Name)
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
			Description: stringValue(tangledRepo.Value.Description),
			CreatedAt:   tangledRepo.Value.CreatedAt,
			RepoDid:     stringValue(tangledRepo.Value.RepoDid),
		})
	}
	return result
}

// canonicalRepoItems returns one record per repository DID. Renamed repositories
// retain old records as aliases; the record keyed by its name is current.
func canonicalRepoItems(items []tangled.Repo) []tangled.Repo {
	result := make([]tangled.Repo, 0, len(items))
	indexesByRepoDID := make(map[string]int)
	for _, repo := range items {
		repoDID := stringValue(repo.Value.RepoDid)
		if repoDID == "" {
			result = append(result, repo)
			continue
		}

		resultIndex, found := indexesByRepoDID[repoDID]
		if !found {
			indexesByRepoDID[repoDID] = len(result)
			result = append(result, repo)
			continue
		}
		if isCanonicalRepoRecord(repo) && !isCanonicalRepoRecord(result[resultIndex]) {
			result[resultIndex] = repo
		}
	}
	return result
}

func isCanonicalRepoRecord(repo tangled.Repo) bool {
	name := stringValue(repo.Value.Name)
	return name != "" && extractRKey(repo.URI) == name
}

// ProvisionRepoInput configures repository provisioning.
type ProvisionRepoInput struct {
	KnotHost    string
	Name        string
	Description string
}

// CreateRepoInput configures provisioning and optional local setup.
type CreateRepoInput struct {
	KnotHost    string
	SSHPort     int
	Name        string
	Description string
	Clone       bool
	PushPath    string
	RemoteName  string
}

// CreateRepo provisions a repository and performs requested local Git setup.
func (s *Service) CreateRepo(ctx context.Context, in CreateRepoInput) (*RepoCreateResult, error) {
	if (in.Clone || in.PushPath != "") && (in.SSHPort < 1 || in.SSHPort > 65535) {
		return nil, fmt.Errorf("SSH port must be between 1 and 65535")
	}
	if in.KnotHost != "" {
		knotHost, err := parseKnotHostname(in.KnotHost)
		if err != nil {
			return nil, err
		}
		in.KnotHost = knotHost
	}
	uri, handle, selectedKnot, warnings, err := s.provisionRepo(ctx, ProvisionRepoInput{
		KnotHost: in.KnotHost, Name: in.Name, Description: in.Description,
	})
	if err != nil {
		return nil, err
	}
	result := &RepoCreateResult{Handle: handle, Name: in.Name, URI: uri, Knot: selectedKnot, Warnings: warnings}
	if in.Clone {
		if _, err := s.CloneRepo(ctx, CloneRepoInput{
			KnotHost: selectedKnot, SSHPort: in.SSHPort,
			Handle: handle, Repo: in.Name, Destination: in.Name,
		}); err != nil {
			return nil, fmt.Errorf("clone new repository: %w", err)
		}
		result.Cloned = true
	}
	if in.PushPath == "" {
		return result, nil
	}
	pushResult, err := s.pushNewRepo(ctx, PushNewRepoInput{
		KnotHost: selectedKnot, SSHPort: in.SSHPort, RepoURI: uri, Dir: in.PushPath,
		Handle: handle, Repo: in.Name, RemoteName: in.RemoteName,
	})
	if pushResult.defaultBranchWarning != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("could not set default branch: %v", pushResult.defaultBranchWarning))
	}
	if err != nil {
		return nil, err
	}
	result.Pushed = true
	result.DefaultBranch = pushResult.defaultBranch
	return result, nil
}

func (s *Service) provisionRepo(ctx context.Context, in ProvisionRepoInput) (uri, handle, knotHost string, warnings []string, err error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return "", "", "", nil, err
	}
	knotHost, warnings, err = s.selectCreationKnot(ctx, atClient, did, in.KnotHost)
	if err != nil {
		return "", "", "", nil, err
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.create")
	if err != nil {
		return "", "", "", nil, err
	}
	repoDid, err := s.knot.New(knotHost, token).CreateRepo(ctx, knot.CreateRepoInput{
		Name: in.Name,
		Rkey: in.Name,
	})
	if err != nil {
		return "", "", "", nil, err
	}
	record := tangledlex.Repo{
		LexiconTypeID: repoCollection,
		Knot:          knotHost,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		RepoDid:       optionalString(repoDid),
	}
	if in.Description != "" {
		record.Description = optionalString(in.Description)
	}
	uri, _, err = atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: repoCollection,
		Rkey:       in.Name,
		Record:     record,
	})
	if err != nil {
		return "", "", "", nil, err
	}
	return uri, s.ownerHandle(ctx, did), knotHost, warnings, nil
}

func (s *Service) selectCreationKnot(ctx context.Context, atClient pdsClient, did, configured string) (string, []string, error) {
	if configured != "" {
		return configured, nil, nil
	}
	page, err := atClient.ListRecords(ctx, did, knotCollection, atproto.ListRecordsOpts{Limit: maxKnotRegistrations + 1})
	if err != nil {
		return "", nil, fmt.Errorf("discover verified Knots: %w", err)
	}
	records := page.Records
	if len(records) == 0 {
		return DefaultKnot, nil, nil
	}
	if len(records) > maxKnotRegistrations {
		return "", nil, fmt.Errorf("found more than %d Knot registrations; select one with --knot or set it in the config file", maxKnotRegistrations)
	}
	hosts := make([]string, 0, len(records))
	warnings := make([]string, 0, len(records))
	seen := make(map[string]bool, len(records))
	for _, record := range records {
		uri, err := syntax.ParseATURI(record.URI)
		if err != nil || uri.Authority().String() != did || uri.Collection().String() != knotCollection || uri.RecordKey().String() == "" {
			warnings = append(warnings, fmt.Sprintf("ignored invalid Knot registration URI %q", record.URI))
			continue
		}
		if err := validateKnotRegistration(record.Value); err != nil {
			warnings = append(warnings, fmt.Sprintf("ignored invalid Knot registration %q: %v", record.URI, err))
			continue
		}
		host, err := parseKnotHostname(uri.RecordKey().String())
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("ignored Knot registration %q: %v", record.URI, err))
			continue
		}
		if seen[host] {
			warnings = append(warnings, fmt.Sprintf("ignored duplicate Knot registration for %s", host))
			continue
		}
		seen[host] = true
		hosts = append(hosts, host)
	}
	verificationErrors := make([]error, len(hosts))
	var verificationGroup sync.WaitGroup
	for index, host := range hosts {
		verificationGroup.Add(1)
		go func() {
			defer verificationGroup.Done()
			verificationErrors[index] = s.knotOwnershipVerifier.Verify(ctx, host, did)
		}()
	}
	verificationGroup.Wait()
	verified := make([]string, 0, len(hosts))
	for index, host := range hosts {
		if err := verificationErrors[index]; err != nil {
			warnings = append(warnings, fmt.Sprintf("could not verify Knot registration for %s: %v", host, err))
			continue
		}
		verified = append(verified, host)
	}
	if len(verified) == 0 {
		return "", nil, fmt.Errorf("no Knot registrations could be verified: %s", strings.Join(warnings, "; "))
	}
	if len(verified) > 1 {
		sort.Strings(verified)
		return "", nil, fmt.Errorf("multiple verified Knots found (%s); select one with --knot or set it in the config file", strings.Join(verified, ", "))
	}
	return verified[0], warnings, nil
}

func validateKnotRegistration(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode record: %w", err)
	}
	var registration knotRegistration
	if err := json.Unmarshal(data, &registration); err != nil {
		return fmt.Errorf("decode record: %w", err)
	}
	if registration.LexiconTypeID != knotCollection {
		return fmt.Errorf("$type must be %q", knotCollection)
	}
	if _, err := syntax.ParseDatetime(registration.CreatedAt); err != nil {
		return fmt.Errorf("invalid createdAt: %w", err)
	}
	return nil
}

func parseKnotHostname(raw string) (string, error) {
	hostname, err := syntax.ParseHandle(raw)
	if err != nil {
		return "", fmt.Errorf("invalid Knot hostname %q: %w", raw, err)
	}
	return hostname.Normalize().String(), nil
}

func (s *Service) ownerHandle(ctx context.Context, did string) string {
	if ident, err := s.resolver.ResolveDID(ctx, did); err == nil {
		return ident.Handle.String()
	}
	return did
}

// SetRepoDefaultBranch sets the default branch of the authenticated user's repo.
func (s *Service) SetRepoDefaultBranch(ctx context.Context, t Target, branch string) (*RepoDefaultBranchResult, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, did)
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

func (s *Service) setDefaultBranchFromDir(ctx context.Context, knotHost, repoURI, dir string) (string, error) {
	atClient, _, err := s.authenticatedPDS(ctx)
	if err != nil {
		return "", err
	}
	branch, err := s.git.CurrentBranch(ctx, dir)
	if err != nil {
		return "", err
	}
	if err := s.setKnotDefaultBranch(ctx, atClient, knotHost, repoURI, branch); err != nil {
		return branch, err
	}
	return branch, nil
}

type pushNewRepoResult struct {
	defaultBranch        string
	defaultBranchWarning error
}

func (s *Service) pushNewRepo(ctx context.Context, in PushNewRepoInput) (pushNewRepoResult, error) {
	branch, defaultBranchErr := s.setDefaultBranchFromDir(ctx, in.KnotHost, in.RepoURI, in.Dir)
	result := pushNewRepoResult{defaultBranch: branch, defaultBranchWarning: defaultBranchErr}
	if err := s.git.PushNewRepo(ctx, gitutil.PushNewRepoParams{
		Dir: in.Dir, KnotHost: in.KnotHost, SSHPort: in.SSHPort,
		Handle: in.Handle, Repo: in.Repo, RemoteName: in.RemoteName,
	}); err != nil {
		return result, fmt.Errorf("push to new repository: %w", err)
	}
	return result, nil
}

// PushNewRepoInput configures pushing a newly created repository.
type PushNewRepoInput struct {
	KnotHost   string
	SSHPort    int
	RepoURI    string
	Dir        string
	Handle     string
	Repo       string
	RemoteName string
}

func (s *Service) setKnotDefaultBranch(ctx context.Context, atClient pdsClient, knotHost, repoURI, branch string) error {
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.setDefaultBranch")
	if err != nil {
		return fmt.Errorf("get knot authorization: %w", err)
	}
	return s.knot.New(knotHost, token).SetDefaultBranch(ctx, knot.SetDefaultBranchInput{
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
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	rkey := extractRKey(repo.URI)
	if err := updateRecord(ctx, atClient, did, repoCollection, rkey, func(value any) (map[string]any, error) {
		record, err := repoRecordMap(value)
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
		return record, nil
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
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.requireOwnedRepo(ctx, t, did)
	if err != nil {
		return nil, err
	}
	if repo.Value.Knot == "" {
		return nil, fmt.Errorf("repo %q has no knot", t.String())
	}
	rkey := extractRKey(repo.URI)
	existingRecord, getErr := atClient.GetRecord(ctx, did, repoCollection, rkey)
	var recordToRestore tangledlex.Repo
	if getErr == nil {
		data, err := json.Marshal(existingRecord.Value)
		if err != nil {
			return nil, fmt.Errorf("encode repository record for restore: %w", err)
		}
		if err := json.Unmarshal(data, &recordToRestore); err != nil {
			return nil, fmt.Errorf("decode repository record for restore: %w", err)
		}
		if err := tangledlex.ValidateRecord(repoCollection, recordToRestore); err != nil {
			return nil, fmt.Errorf("validate repository record for restore: %w", err)
		}
	}
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
	if err := s.knot.New(repo.Value.Knot, token).DeleteRepo(ctx, knot.DeleteRepoInput{
		DID:  did,
		Name: t.Repo,
		Rkey: rkey,
	}); err != nil {
		if getErr == nil {
			if _, _, restoreErr := atClient.PutRecord(ctx, atproto.PutRecordInput{
				Repo: did, Collection: repoCollection, Rkey: rkey, Record: recordToRestore,
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
	atClient, ownerDID, err := s.authenticatedPDS(ctx)
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
	repoDID, err := s.knot.New(src.Knot, token).CreateRepo(ctx, knot.CreateRepoInput{
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
			Knot:          src.Knot,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			RepoDid:       optionalString(repoDID),
			Source:        optionalString(src.URI),
		},
	})
	if err != nil {
		cleanupErr := s.deleteFork(ctx, atClient, src.Knot, ownerDID, name)
		if cleanupErr != nil {
			return nil, fmt.Errorf("write fork record: %w; delete orphaned fork: %v", err, cleanupErr)
		}
		return nil, fmt.Errorf("write fork record: %w", err)
	}
	return &RepoForkResult{Handle: s.ownerHandle(ctx, ownerDID), Name: name, URI: uri, Knot: src.Knot}, nil
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
	ident, err := s.resolver.ResolveHandle(ctx, t.Handle)
	if err != nil {
		return forkSource{}, fmt.Errorf("resolve handle %q: %w", t.Handle, err)
	}
	uri := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, t.Repo)
	repo, err := s.appview.GetRepo(ctx, uri)
	if err != nil {
		return forkSource{}, fmt.Errorf("get source repository %s: %w", t, err)
	}
	if repo.Value.Knot == "" {
		return forkSource{}, fmt.Errorf("source repository %s has no knot", t)
	}
	if stringValue(repo.Value.RepoDid) == "" {
		return forkSource{}, fmt.Errorf("source repository %s has no repo DID", t)
	}
	if repo.URI != "" {
		uri = repo.URI
	}
	return forkSource{URI: uri, Knot: repo.Value.Knot, RepoDID: stringValue(repo.Value.RepoDid)}, nil
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
