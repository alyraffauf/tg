package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type provisionRepoInput struct {
	KnotHost    string
	Name        string
	Description string
}

type provisionedRepo struct {
	URI      string
	RepoDID  string
	Handle   string
	KnotHost string
	Warnings []string
}

// CreateRepoInput configures provisioning and optional local setup.
type CreateRepoInput struct {
	KnotHost      string
	SSHPort       int
	Name          string
	Description   string
	Clone         bool
	CloneProtocol string
	PushPath      string
	RemoteName    string
}

// CreateRepo provisions a repository and performs requested local Git setup.
func (s *Service) CreateRepo(ctx context.Context, in CreateRepoInput) (*RepoCreateResult, error) {
	if in.Clone {
		cloneProtocol, err := validateCloneProtocol(in.CloneProtocol)
		if err != nil {
			return nil, err
		}
		in.CloneProtocol = cloneProtocol
	}
	directKnot := in.KnotHost != ""
	if directKnot && ((in.Clone && in.CloneProtocol == "ssh") || in.PushPath != "") && (in.SSHPort < 1 || in.SSHPort > 65535) {
		return nil, fmt.Errorf("SSH port must be between 1 and 65535")
	}
	if in.KnotHost != "" {
		knotHost, err := parseKnotHostname(in.KnotHost)
		if err != nil {
			return nil, err
		}
		in.KnotHost = knotHost
	}
	provisioned, err := s.provisionRepo(ctx, provisionRepoInput{
		KnotHost: in.KnotHost, Name: in.Name, Description: in.Description,
	})
	if err != nil {
		return nil, err
	}
	result := &RepoCreateResult{
		Handle: provisioned.Handle, Name: in.Name, URI: provisioned.URI,
		Knot: provisioned.KnotHost, Warnings: provisioned.Warnings,
	}
	remoteKnotHost := ""
	remoteSSHPort := 22
	if directKnot {
		remoteKnotHost = provisioned.KnotHost
		remoteSSHPort = in.SSHPort
	}
	if in.Clone {
		if _, err := s.cloneResolvedRepo(ctx, resolvedCloneRepoInput{
			KnotHost: remoteKnotHost, SSHPort: remoteSSHPort,
			Protocol: in.CloneProtocol, Handle: provisioned.Handle, Repo: in.Name, RepoDID: provisioned.RepoDID, Destination: in.Name,
		}); err != nil {
			return nil, fmt.Errorf("clone new repository: %w", err)
		}
		result.Cloned = true
	}
	if in.PushPath == "" {
		return result, nil
	}
	pushResult, err := s.pushNewRepo(ctx, PushNewRepoInput{
		KnotHost: provisioned.KnotHost, RemoteKnotHost: remoteKnotHost, SSHPort: remoteSSHPort,
		RepoDID: provisioned.RepoDID, Dir: in.PushPath,
		RemoteName: in.RemoteName,
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

func (s *Service) provisionRepo(ctx context.Context, in provisionRepoInput) (*provisionedRepo, error) {
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	knotHost, warnings, err := s.selectCreationKnot(ctx, atClient, did, in.KnotHost)
	if err != nil {
		return nil, err
	}
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+knotHost, "sh.tangled.repo.create")
	if err != nil {
		return nil, err
	}
	repoDID, err := s.knot.New(knotHost, token).CreateRepo(ctx, knot.CreateRepoInput{
		Name: in.Name,
		Rkey: in.Name,
	})
	if err != nil {
		return nil, err
	}
	record := tangledlex.Repo{
		LexiconTypeID: repoCollection,
		Knot:          knotHost,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		RepoDid:       optionalString(repoDID),
	}
	if in.Description != "" {
		record.Description = optionalString(in.Description)
	}
	uri, _, err := atClient.PutRecord(ctx, atproto.PutRecordInput{
		Repo:       did,
		Collection: repoCollection,
		Rkey:       in.Name,
		Record:     record,
	})
	if err != nil {
		return nil, err
	}
	return &provisionedRepo{
		URI: uri, RepoDID: repoDID, Handle: s.ownerHandle(ctx, did),
		KnotHost: knotHost, Warnings: warnings,
	}, nil
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

func (s *Service) setDefaultBranchFromDir(ctx context.Context, knotHost, repoDID, dir string) (string, error) {
	atClient, _, err := s.authenticatedPDS(ctx)
	if err != nil {
		return "", err
	}
	branch, err := s.git.CurrentBranch(ctx, dir)
	if err != nil {
		return "", err
	}
	if err := s.setKnotDefaultBranch(ctx, atClient, knotHost, repoDID, branch); err != nil {
		return branch, err
	}
	return branch, nil
}

type pushNewRepoResult struct {
	defaultBranch        string
	defaultBranchWarning error
}

// PushNewRepoInput configures pushing a newly created repository.
type PushNewRepoInput struct {
	KnotHost       string
	RemoteKnotHost string
	SSHPort        int
	RepoDID        string
	Dir            string
	RemoteName     string
}

func (s *Service) pushNewRepo(ctx context.Context, in PushNewRepoInput) (pushNewRepoResult, error) {
	branch, defaultBranchErr := s.setDefaultBranchFromDir(ctx, in.KnotHost, in.RepoDID, in.Dir)
	result := pushNewRepoResult{defaultBranch: branch, defaultBranchWarning: defaultBranchErr}
	if err := s.git.PushNewRepo(ctx, gitutil.PushNewRepoParams{
		Dir: in.Dir, KnotHost: in.RemoteKnotHost, SSHPort: in.SSHPort,
		RepoDID: in.RepoDID, RemoteName: in.RemoteName,
	}); err != nil {
		return result, fmt.Errorf("push to new repository: %w", err)
	}
	return result, nil
}
