package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type provisionRepoInput struct {
	Source             string
	ContributionPolicy string
	KnotHost           string
	Name               string
	Description        string
}

type provisionedRepo struct {
	URI      string
	RepoDID  string
	Handle   string
	KnotHost string
	Warnings []string
	Declared bool
}

// CreateRepoInput configures provisioning and optional local setup.
type CreateRepoInput struct {
	ContributionPolicy string
	KnotHost           string
	SSHPort            int
	Name               string
	Description        string
	Clone              bool
	CloneProtocol      string
	PushPath           string
	RemoteName         string
}

// CreateRepo provisions a repository and performs requested local Git setup.
func (s *Service) CreateRepo(ctx context.Context, in CreateRepoInput) (*RepoCreateResult, error) {
	if _, err := syntax.ParseRecordKey(in.Name); err != nil {
		return nil, fmt.Errorf("invalid repository name %q: %w", in.Name, err)
	}
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
		KnotHost: in.KnotHost, Name: in.Name, Description: in.Description, ContributionPolicy: in.ContributionPolicy,
	})
	if provisioned == nil {
		return nil, err
	}
	result := &RepoCreateResult{
		RepoDID: provisioned.RepoDID, Declared: provisioned.Declared,
		Handle: provisioned.Handle, Name: in.Name, URI: provisioned.URI,
		Knot: provisioned.KnotHost, Warnings: provisioned.Warnings,
	}
	if err != nil {
		return result, err
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
			return result, fmt.Errorf("repository %s was created, but cloning failed: %w", result.URI, err)
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
		return result, fmt.Errorf("repository %s was created, but pushing failed: %w", result.URI, err)
	}
	result.Pushed = true
	result.DefaultBranch = pushResult.defaultBranch
	return result, nil
}

func (s *Service) provisionRepo(ctx context.Context, in provisionRepoInput) (*provisionedRepo, error) {
	if err := validateContributionPolicy(in.ContributionPolicy); err != nil {
		return nil, err
	}
	pds, actor, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	host := in.KnotHost
	if host == "" {
		host = DefaultKnot
	}
	key := strings.ToLower(in.Name)
	if _, err := pds.GetRecord(ctx, actor, tangled.DeclarationCollection, key); err == nil {
		return nil, fmt.Errorf("repository declaration %q already exists", key)
	} else if !isRecordNotFound(err) {
		return nil, fmt.Errorf("check repository declaration: %w", err)
	}
	token, err := pds.GetServiceAuth(ctx, serviceDID(host), "org.tangled.repo.createRepo")
	if err != nil {
		return nil, err
	}
	did, err := s.knot.New(host, token).CreateRepo(ctx, knot.CreateRepoInput{Name: in.Name, Description: in.Description, Source: in.Source})
	if err != nil {
		return nil, err
	}
	provisioned := &provisionedRepo{URI: "at://" + did + "/" + tangled.RepoCollection + "/self", RepoDID: did, Handle: s.ownerHandle(ctx, actor), KnotHost: host}
	if in.ContributionPolicy != "" {
		repo := &tangled.Repo{Value: tangled.RepoMetadata{Knot: host, RepoDid: &did, Name: &in.Name}}
		policyWarnings, err := s.editManifest(ctx, repo, func(record map[string]any) error { record["contributionPolicy"] = in.ContributionPolicy; return nil })
		if err != nil {
			if isDefiniteRejection(err) {
				if cleanup := s.deleteProvisionedRepo(ctx, pds, host, knot.DeleteRepoInput{Repo: did}); cleanup != nil {
					return provisioned, fmt.Errorf("repository %s policy was rejected: %w; cleanup failed: %v", did, err, cleanup)
				}
				return nil, fmt.Errorf("repository %s policy was rejected and the repository was deleted: %w", did, err)
			}
			return provisioned, fmt.Errorf("repository %s was created without a declaration; policy setup could not be confirmed: %w; inspect it before retrying", did, err)
		}
		provisioned.Warnings = append(provisioned.Warnings, policyWarnings...)
	}
	record := map[string]any{"$type": tangled.DeclarationCollection, "repo": did, "createdAt": nowString()}
	uri, cid, err := pds.CreateRecord(ctx, atproto.PutRecordInput{Repo: actor, Collection: tangled.DeclarationCollection, Rkey: key, Record: record})
	if err != nil {
		if isDefiniteRejection(err) {
			cleanup := s.deleteProvisionedRepo(ctx, pds, host, knot.DeleteRepoInput{Repo: did})
			if cleanup != nil {
				return provisioned, fmt.Errorf("write declaration: %w; repository %s cleanup failed: %v; inspect it before retrying", err, did, cleanup)
			}
			return nil, fmt.Errorf("write declaration: %w; repository %s was deleted", err, did)
		}
		return provisioned, fmt.Errorf("write declaration for repository %s: %w; inspect it before retrying", did, err)
	}
	provisioned.Declared = true
	indexCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.appview.AwaitRecord(indexCtx, uri, cid); err != nil {
		provisioned.Warnings = append(provisioned.Warnings, fmt.Sprintf("repository saved, but indexing is pending: %v", err))
	}
	return provisioned, nil
}
func (s *Service) deleteProvisionedRepo(ctx context.Context, pds pdsClient, host string, in knot.DeleteRepoInput) error {
	token, err := pds.GetServiceAuth(ctx, serviceDID(host), "org.tangled.repo.deleteRepo")
	if err != nil {
		return err
	}
	return s.knot.New(host, token).DeleteRepo(ctx, in)
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
