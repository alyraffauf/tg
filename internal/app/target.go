package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Target identifies a repository by owner handle (or DID) and repo name.
type Target struct {
	Handle  string
	Repo    string
	RepoDID string
	// ownerDID preserves the verified owner when a repository DID was resolved.
	ownerDID string
}

func (t Target) String() string { return t.Handle + "/" + t.Repo }

// ParseTarget parses a "handle/repo" (or "did:plc:.../repo") argument.
func ParseTarget(arg string) (Target, error) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(parts[1], "/") {
		return Target{}, fmt.Errorf("expected handle/repo, got %q", arg)
	}
	return Target{Handle: parts[0], Repo: parts[1]}, nil
}

// TargetFromCWD detects the target using the service's Git client.
func (s *Service) TargetFromCWD(ctx context.Context) (Target, error) {
	target, _, err := s.targetFromCWD(ctx, false)
	return target, err
}

func (s *Service) repoFromCWD(ctx context.Context) (Target, *tangled.Repo, error) {
	return s.targetFromCWD(ctx, true)
}

func (s *Service) targetFromCWD(ctx context.Context, resolveHosted bool) (Target, *tangled.Repo, error) {
	candidates, err := s.git.DetectRepoCandidatesFromCWD(ctx)
	if err != nil {
		return Target{}, nil, fmt.Errorf("detect repo from current directory: %w", err)
	}
	var failures []string
	for _, candidate := range candidates {
		if candidate.RepoDID != "" {
			target, repo, err := s.resolveRepoDID(ctx, candidate.RepoDID, candidate.KnotHost)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", candidate.RepoDID, err))
				continue
			}
			return target, repo, nil
		}
		target := Target{Handle: candidate.Handle, Repo: candidate.Repo}
		if candidate.KnotHost == "" && !resolveHosted {
			return target, nil, nil
		}
		repo, err := s.resolveRepo(ctx, target)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", target, err))
			continue
		}
		if candidate.KnotHost != "" {
			candidateHost, candidateErr := knotAuthorityFromHost(candidate.KnotHost)
			recordHost, recordErr := knotAuthorityFromHost(repo.Value.Knot)
			if candidateErr != nil || recordErr != nil || candidateHost != recordHost {
				failures = append(failures, fmt.Sprintf("%s: remote Knot %q does not match repository record Knot %q", target, candidate.KnotHost, repo.Value.Knot))
				continue
			}
		}
		return target, repo, nil
	}
	return Target{}, nil, fmt.Errorf("no Git remote matches a Tangled repository record: %s; pass the repository as handle/repo", strings.Join(failures, "; "))
}

func (s *Service) resolveRepoDID(ctx context.Context, did, remoteHost string) (Target, *tangled.Repo, error) {
	repo, err := s.resolveRepo(ctx, Target{RepoDID: did})
	if err != nil {
		return Target{}, nil, err
	}
	endpoint, err := parseKnotServiceEndpoint(repo.RecordURL)
	if err != nil {
		return Target{}, nil, err
	}
	if remoteHost != "" {
		host, e := knotAuthorityFromHost(remoteHost)
		if e != nil || host != endpoint.Authority {
			return Target{}, nil, fmt.Errorf("remote Knot %q does not match repository DID Knot %q", remoteHost, endpoint.Authority)
		}
	}
	target, err := s.targetFromRepoDIDRecord(ctx, did, endpoint.Authority, repo)
	return target, repo, err
}
func repositoryKnotServiceURL(ident *identity.Identity) (string, error) {
	if ident == nil {
		return "", errors.New("missing repository identity")
	}
	service := ident.Services["atproto_pds"]
	if service.Type != "AtprotoPersonalDataServer" || service.URL == "" {
		return "", errors.New("repository DID has no AtprotoPersonalDataServer endpoint")
	}
	return service.URL, nil
}

type knotServiceEndpoint struct {
	Hostname  string
	Authority string
}

func parseKnotServiceEndpoint(raw string) (knotServiceEndpoint, error) {
	serviceURL, err := url.Parse(raw)
	if err != nil {
		return knotServiceEndpoint{}, fmt.Errorf("parse service endpoint %q: %w", raw, err)
	}
	path := serviceURL.EscapedPath()
	if serviceURL.Scheme != "https" ||
		serviceURL.Hostname() == "" ||
		serviceURL.User != nil ||
		serviceURL.RawQuery != "" ||
		serviceURL.ForceQuery ||
		serviceURL.Fragment != "" ||
		(path != "" && path != "/") {
		return knotServiceEndpoint{}, fmt.Errorf("repository DID service endpoint must be an HTTPS Knot URL, got %q", raw)
	}
	hostname, err := parseKnotHostname(serviceURL.Hostname())
	if err != nil {
		return knotServiceEndpoint{}, err
	}
	port := serviceURL.Port()
	if port == "" {
		return knotServiceEndpoint{Hostname: hostname, Authority: hostname}, nil
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return knotServiceEndpoint{}, fmt.Errorf("repository DID service endpoint has invalid HTTPS port %q", port)
	}
	if portNumber == 443 {
		return knotServiceEndpoint{Hostname: hostname, Authority: hostname}, nil
	}
	return knotServiceEndpoint{Hostname: hostname, Authority: hostname + ":" + strconv.Itoa(portNumber)}, nil
}

func knotHostnameFromHost(raw string) (string, error) {
	hostURL, err := url.Parse("https://" + raw)
	if err != nil || hostURL.Host != raw || hostURL.Hostname() == "" {
		return "", fmt.Errorf("invalid Knot host %q", raw)
	}
	return parseKnotHostname(hostURL.Hostname())
}

func knotAuthorityFromHost(raw string) (string, error) {
	endpoint, err := parseKnotServiceEndpoint("https://" + raw)
	if err != nil {
		return "", err
	}
	return endpoint.Authority, nil
}

func (s *Service) targetFromRepoDIDRecord(ctx context.Context, did, host string, repo *tangled.Repo) (Target, error) {
	if repo == nil || repo.OwnerDID == "" || stringValue(repo.Value.RepoDid) != did {
		return Target{}, fmt.Errorf("repository metadata does not match DID %q", did)
	}
	if repo.Value.Knot != host {
		return Target{}, fmt.Errorf("repository Knot %q does not match DID Knot %q", repo.Value.Knot, host)
	}
	return Target{Handle: s.ownerHandle(ctx, repo.OwnerDID), Repo: stringValue(repo.Value.Name), RepoDID: did, ownerDID: repo.OwnerDID}, nil
}
func (s *Service) resolveRepo(ctx context.Context, t Target) (*tangled.Repo, error) {
	var repo *tangled.Repo
	var err error
	var expectedOwner string
	if t.RepoDID != "" {
		repo, err = s.appview.GetRepoByDID(ctx, t.RepoDID)
	} else {
		owner, e := s.targetOwnerDID(ctx, t)
		if e != nil {
			return nil, e
		}
		expectedOwner = owner
		repo, err = s.appview.GetRepo(ctx, "at://"+owner+"/"+tangled.DeclarationCollection+"/"+strings.ToLower(t.Repo))
	}
	if err != nil {
		return nil, err
	}
	if repo == nil || stringValue(repo.Value.RepoDid) == "" {
		return nil, errors.New("appview returned no repository DID")
	}
	if expectedOwner != "" && (repo.OwnerDID != expectedOwner || !strings.EqualFold(stringValue(repo.Value.Name), t.Repo)) {
		return nil, errors.New("appview returned a different repository declaration")
	}
	did := stringValue(repo.Value.RepoDid)
	if t.RepoDID != "" && t.RepoDID != did {
		return nil, errors.New("appview returned a different repository DID")
	}
	ident, err := s.resolver.ResolveDID(ctx, did)
	if err != nil {
		return nil, fmt.Errorf("resolve repository DID: %w", err)
	}
	recordURL, err := repositoryKnotServiceURL(ident)
	if err != nil {
		return nil, err
	}
	endpoint, err := parseKnotServiceEndpoint(recordURL)
	if err != nil {
		return nil, err
	}
	gitService, hasGitService := ident.Services["tangled_knot"]
	gitEndpoint := gitService.URL
	if !hasGitService {
		// Legacy repository DIDs identify their Knot only through atproto_pds.
		gitEndpoint = "https://" + endpoint.Authority + "/" + did
	} else if gitService.Type != "TangledKnot" || gitEndpoint == "" {
		return nil, errors.New("repository DID has no TangledKnot endpoint")
	}
	gitURL, err := url.Parse(gitEndpoint)
	if err != nil || gitURL.Scheme != "https" || gitURL.User != nil || gitURL.RawQuery != "" || gitURL.ForceQuery || gitURL.Fragment != "" {
		return nil, fmt.Errorf("invalid or mismatched repository Git endpoint %q", gitEndpoint)
	}
	gitAuthority, err := knotAuthorityFromHost(gitURL.Host)
	if err != nil || gitAuthority != endpoint.Authority {
		return nil, fmt.Errorf("invalid or mismatched repository Git endpoint %q", gitEndpoint)
	}
	if repo.Value.Knot != "" && repo.Value.Knot != endpoint.Hostname && repo.Value.Knot != endpoint.Authority {
		return nil, errors.New("Bobbin Knot does not match repository DID")
	}
	repo.RecordURL = "https://" + endpoint.Authority
	repo.GitURL = gitURL.String()
	repo.Value.Knot = endpoint.Authority
	if _, err := syntax.ParseDID(repo.OwnerDID); err != nil {
		return nil, fmt.Errorf("invalid repository owner: %w", err)
	}
	repo.DeclarationURI = "at://" + repo.OwnerDID + "/" + tangled.DeclarationCollection + "/" + strings.ToLower(stringValue(repo.Value.Name))
	return repo, nil
}

func (s *Service) verifyRepositoryClaim(ctx context.Context, repo *tangled.Repo) error {
	did := stringValue(repo.Value.RepoDid)
	pds, owner, err := s.publicPDS(ctx, repo.OwnerDID)
	if err != nil {
		return err
	}
	claim, err := pds.GetRecord(ctx, owner, tangled.DeclarationCollection, strings.ToLower(stringValue(repo.Value.Name)))
	if err != nil {
		return fmt.Errorf("verify repository declaration: %w", err)
	}
	var declaration struct {
		Repo string `json:"repo"`
	}
	data, err := json.Marshal(claim.Value)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &declaration); err != nil {
		return err
	}
	if declaration.Repo != did {
		return errors.New("owner declaration does not point back to repository DID")
	}
	return nil
}

func (s *Service) resolveVerifiedRepo(ctx context.Context, t Target) (*tangled.Repo, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	if err := s.verifyRepositoryClaim(ctx, repo); err != nil {
		return nil, err
	}
	return repo, nil
}
func (s *Service) targetOwnerDID(ctx context.Context, t Target) (string, error) {
	if t.ownerDID != "" {
		return t.ownerDID, nil
	}
	if did, err := syntax.ParseDID(t.Handle); err == nil {
		return did.String(), nil
	}
	ident, err := s.resolver.ResolveHandle(ctx, t.Handle)
	if err != nil {
		return "", err
	}
	return ident.DID.String(), nil
}
func (s *Service) repoDID(ctx context.Context, t Target) (string, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return "", err
	}
	return stringValue(repo.Value.RepoDid), nil
}
func (s *Service) requireOwnedRepo(ctx context.Context, t Target, did string) (*tangled.Repo, error) {
	repo, err := s.resolveVerifiedRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	if repo.OwnerDID != did {
		return nil, fmt.Errorf("repository %q is not owned by the active account", t.String())
	}
	return repo, nil
}
func isDescribeRepoUnsupported(err error) bool {
	var e *atclient.APIError
	return errors.As(err, &e) && ((e.StatusCode == http.StatusNotFound && (e.Name == "XRPCNotSupported" || e.Name == "MethodNotFound")) || (e.StatusCode == http.StatusNotImplemented && e.Name == "MethodNotImplemented"))
}

func (s *Service) repositoryHost(ctx context.Context, did string) (string, error) {
	ident, err := s.resolver.ResolveDID(ctx, did)
	if err != nil {
		return "", err
	}
	raw, err := repositoryKnotServiceURL(ident)
	if err != nil {
		return "", err
	}
	endpoint, err := parseKnotServiceEndpoint(raw)
	if err != nil {
		return "", err
	}
	return endpoint.Authority, nil
}
