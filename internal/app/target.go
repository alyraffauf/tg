package app

import (
	"context"
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
			candidateHost, candidateErr := parseKnotHostname(candidate.KnotHost)
			recordHost, recordErr := parseKnotHostname(repo.Value.Knot)
			if candidateErr != nil || recordErr != nil || candidateHost != recordHost {
				failures = append(failures, fmt.Sprintf("%s: remote Knot %q does not match repository record Knot %q", target, candidate.KnotHost, repo.Value.Knot))
				continue
			}
		}
		return target, repo, nil
	}
	return Target{}, nil, fmt.Errorf("no Git remote matches a Tangled repository record: %s; pass the repository as handle/repo", strings.Join(failures, "; "))
}

// resolveRepoDID maps a repository DID to its current owner record and verifies
// that the DID document, Knot metadata, and ATProto record agree.
func (s *Service) resolveRepoDID(ctx context.Context, repoDID, remoteKnotHost string) (Target, *tangled.Repo, error) {
	if _, err := syntax.ParseDID(repoDID); err != nil {
		return Target{}, nil, fmt.Errorf("invalid repository DID %q: %w", repoDID, err)
	}
	ident, err := s.resolver.ResolveDID(ctx, repoDID)
	if err != nil {
		return Target{}, nil, fmt.Errorf("resolve Knot for repository DID %q: %w", repoDID, err)
	}
	serviceURL, err := repositoryKnotServiceURL(ident)
	if err != nil {
		return Target{}, nil, fmt.Errorf("resolve Knot for repository DID %q: %w", repoDID, err)
	}
	knotEndpoint, err := parseKnotServiceEndpoint(serviceURL)
	if err != nil {
		return Target{}, nil, fmt.Errorf("resolve Knot for repository DID %q: %w", repoDID, err)
	}
	if remoteKnotHost != "" {
		remoteHostname, parseErr := knotHostnameFromHost(remoteKnotHost)
		if parseErr != nil || remoteHostname != knotEndpoint.Hostname {
			return Target{}, nil, fmt.Errorf("remote Knot %q does not match repository DID Knot %q", remoteKnotHost, knotEndpoint.Authority)
		}
	}

	description, describeErr := s.knot.NewPublic(knotEndpoint.Authority).DescribeRepo(ctx, repoDID)
	if describeErr != nil {
		if !isDescribeRepoUnsupported(describeErr) {
			return Target{}, nil, fmt.Errorf("describe repository DID %q through Knot %q: %w", repoDID, knotEndpoint.Authority, describeErr)
		}
		repo, appviewErr := s.appview.GetRepoByDID(ctx, repoDID)
		if appviewErr != nil {
			return Target{}, nil, fmt.Errorf("find repository DID %q through appview after Knot %q returned %v: %w", repoDID, knotEndpoint.Authority, describeErr, appviewErr)
		}
		target, err := s.targetFromRepoDIDRecord(ctx, repoDID, knotEndpoint.Hostname, repo)
		return target, repo, err
	}
	if description.RepoDID != repoDID {
		return Target{}, nil, fmt.Errorf("Knot described repository DID %q as %q", repoDID, description.RepoDID)
	}
	if _, err := syntax.ParseDID(description.OwnerDID); err != nil {
		return Target{}, nil, fmt.Errorf("Knot returned invalid owner DID %q for repository %q: %w", description.OwnerDID, repoDID, err)
	}
	if _, err := syntax.ParseRecordKey(description.RKey); err != nil {
		return Target{}, nil, fmt.Errorf("Knot returned invalid repository record key %q for %q: %w", description.RKey, repoDID, err)
	}

	recordURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", description.OwnerDID, description.RKey)
	repo, err := s.appview.GetRepo(ctx, recordURI)
	if err != nil {
		return Target{}, nil, fmt.Errorf("get repository record %q: %w", recordURI, err)
	}
	if repo.URI == "" {
		repo.URI = recordURI
	}
	target, err := s.targetFromRepoDIDRecord(ctx, repoDID, knotEndpoint.Hostname, repo)
	if err != nil {
		return Target{}, nil, err
	}
	if target.ownerDID != description.OwnerDID || extractRKey(repo.URI) != description.RKey {
		return Target{}, nil, fmt.Errorf("repository record %q does not match Knot owner %q and record key %q", repo.URI, description.OwnerDID, description.RKey)
	}
	return target, repo, nil
}

func isDescribeRepoUnsupported(err error) bool {
	var apiError *atclient.APIError
	if !errors.As(err, &apiError) {
		return false
	}
	switch apiError.StatusCode {
	case http.StatusNotFound:
		return apiError.Name == "XRPCNotSupported"
	case http.StatusNotImplemented:
		return apiError.Name == "MethodNotImplemented"
	default:
		return false
	}
}

func repositoryKnotServiceURL(ident *identity.Identity) (string, error) {
	if ident == nil {
		return "", errors.New("nil identity has no Knot service endpoint")
	}
	services := []struct {
		id          string
		serviceType string
	}{
		{id: "tangled_knot", serviceType: "TangledKnot"},
		{id: "atproto_pds", serviceType: "AtprotoPersonalDataServer"},
	}
	for _, expected := range services {
		service := ident.Services[expected.id]
		if service.Type == expected.serviceType && service.URL != "" {
			return service.URL, nil
		}
	}
	return "", errors.New("DID document has no TangledKnot or AtprotoPersonalDataServer service endpoint")
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

func (s *Service) targetFromRepoDIDRecord(ctx context.Context, repoDID, knotHostname string, repo *tangled.Repo) (Target, error) {
	if repo == nil {
		return Target{}, errors.New("appview returned an empty repository record")
	}
	uri, err := syntax.ParseATURI(repo.URI)
	if err != nil || uri.Collection().String() != repoCollection || uri.RecordKey().String() == "" {
		return Target{}, fmt.Errorf("appview returned invalid repository record URI %q", repo.URI)
	}
	ownerDID := uri.Authority().String()
	if _, err := syntax.ParseDID(ownerDID); err != nil {
		return Target{}, fmt.Errorf("repository record %q has invalid owner DID %q: %w", repo.URI, ownerDID, err)
	}
	if recordRepoDID := stringValue(repo.Value.RepoDid); recordRepoDID != repoDID {
		return Target{}, fmt.Errorf("repository record %q has repository DID %q, want %q", repo.URI, recordRepoDID, repoDID)
	}
	recordKnotHost, err := parseKnotHostname(repo.Value.Knot)
	if err != nil || recordKnotHost != knotHostname {
		return Target{}, fmt.Errorf("repository record Knot %q does not match repository DID Knot %q", repo.Value.Knot, knotHostname)
	}
	repoName := stringValue(repo.Value.Name)
	if repoName == "" {
		repoName = uri.RecordKey().String()
	}
	return Target{
		Handle:   s.ownerHandle(ctx, ownerDID),
		Repo:     repoName,
		RepoDID:  repoDID,
		ownerDID: ownerDID,
	}, nil
}

// resolveRepo finds a repository record even when its rkey does not match
// the repository name.
func (s *Service) resolveRepo(ctx context.Context, t Target) (*tangled.Repo, error) {
	if t.RepoDID != "" {
		repo, err := s.appview.GetRepoByDID(ctx, t.RepoDID)
		if err != nil {
			return nil, fmt.Errorf("get repository by DID %q: %w", t.RepoDID, err)
		}
		if recordRepoDID := stringValue(repo.Value.RepoDid); recordRepoDID != t.RepoDID {
			return nil, fmt.Errorf("repository record %q has repository DID %q, want %q", repo.URI, recordRepoDID, t.RepoDID)
		}
		return repo, nil
	}
	ownerDID, err := s.targetOwnerDID(ctx, t)
	if err != nil {
		return nil, err
	}

	recordURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ownerDID, t.Repo)
	if repo, err := s.appview.GetRepo(ctx, recordURI); err == nil {
		if repo.URI == "" {
			repo.URI = recordURI
		}
		if isCanonicalRepoRecord(*repo) || stringValue(repo.Value.Name) == "" {
			return repo, nil
		}
		return s.resolveCanonicalRepo(ctx, ownerDID, t, repo)
	} else if !shouldListRepoRecords(err) {
		return nil, fmt.Errorf("get repository %q: %w", t.Repo, err)
	}

	return s.resolveCanonicalRepo(ctx, ownerDID, t, nil)
}

func (s *Service) targetOwnerDID(ctx context.Context, target Target) (string, error) {
	if target.ownerDID != "" {
		did, err := syntax.ParseDID(target.ownerDID)
		if err != nil {
			return "", fmt.Errorf("invalid owner DID %q: %w", target.ownerDID, err)
		}
		return did.String(), nil
	}
	if did, err := syntax.ParseDID(target.Handle); err == nil {
		return did.String(), nil
	}

	ident, err := s.resolver.ResolveHandle(ctx, target.Handle)
	if err != nil {
		return "", fmt.Errorf("resolve handle %q: %w", target.Handle, err)
	}
	return ident.DID.String(), nil
}

func (s *Service) resolveCanonicalRepo(ctx context.Context, ownerDID string, t Target, directRepo *tangled.Repo) (*tangled.Repo, error) {
	repos, err := s.appview.ListRepos(ctx, ownerDID)
	if err != nil {
		return nil, fmt.Errorf("list repos for %q: %w", t.Handle, err)
	}
	if directRepo != nil {
		return canonicalRepoForAlias(repos.Items, *directRepo), nil
	}
	for _, repo := range repos.Items {
		if stringValue(repo.Value.Name) == t.Repo || extractRKey(repo.URI) == t.Repo {
			return canonicalRepoForAlias(repos.Items, repo), nil
		}
	}
	return nil, fmt.Errorf("repo %q not found for handle %q", t.Repo, t.Handle)
}

func canonicalRepoForAlias(items []tangled.Repo, alias tangled.Repo) *tangled.Repo {
	aliasName := stringValue(alias.Value.Name)
	aliasRepoDID := stringValue(alias.Value.RepoDid)
	if aliasName == "" || aliasRepoDID == "" {
		return &alias
	}
	for index := range items {
		candidate := &items[index]
		if stringValue(candidate.Value.RepoDid) == aliasRepoDID && isCanonicalRepoRecord(*candidate) && stringValue(candidate.Value.Name) == aliasName {
			return candidate
		}
	}
	return &alias
}

// repoDID resolves a target to the key used by issue and pull-request listings.
func (s *Service) repoDID(ctx context.Context, t Target) (string, error) {
	record, err := s.resolveRepo(ctx, t)
	if err != nil {
		return "", err
	}
	repoDID := stringValue(record.Value.RepoDid)
	if repoDID == "" {
		return "", fmt.Errorf("repository %q has no repository DID", t.String())
	}
	return repoDID, nil
}

// requireOwnedRepo resolves a target and verifies it is owned by did.
func (s *Service) requireOwnedRepo(ctx context.Context, t Target, did string) (*tangled.Repo, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	if extractDID(repo.URI) != did {
		return nil, fmt.Errorf("repo %q is not owned by the authenticated user", t.String())
	}
	return repo, nil
}

// shouldListRepoRecords reports whether err indicates the record is absent at
// the name-derived rkey and so a full listing is required to find it.
func shouldListRepoRecords(err error) bool {
	var apiError *atclient.APIError
	if !errors.As(err, &apiError) {
		return false
	}
	if apiError.StatusCode == http.StatusNotFound {
		return true
	}

	// Bobbin wraps an upstream PDS 400 as a 502 when no record exists at the
	// name-derived rkey. Listing is required to find the record's actual rkey.
	return apiError.StatusCode == http.StatusBadGateway &&
		apiError.Name == "UpstreamFailed" &&
		strings.Contains(apiError.Message, "upstream returned status 400 Bad Request")
}
