package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/internal/gitutil"
)

// GitPushToken returns credentials for the hosted Git proxy or the current
// repository's recorded Knot.
func (s *Service) GitPushToken(ctx context.Context, requestedHost string) (*GitCredentialResult, error) {
	_, repo, err := s.repoFromCWD(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.verifyRepositoryClaim(ctx, repo); err != nil {
		return nil, err
	}
	host, err := knotAuthorityFromHost(repo.Value.Knot)
	if err != nil {
		return nil, err
	}
	requestedHost = strings.TrimSpace(requestedHost)
	requestedAuthority, err := knotAuthorityFromHost(requestedHost)
	if err != nil {
		return &GitCredentialResult{}, nil
	}
	if requestedAuthority != gitutil.HostedGitHost && requestedAuthority != host {
		return &GitCredentialResult{}, nil
	}
	hasPushScope, isOAuth, err := s.sessions.OAuthSessionHasScope(ctx, "rpc:org.tangled.git.receivePack?aud=*")
	if err != nil {
		return nil, fmt.Errorf("check OAuth push permission: %w", err)
	}
	if isOAuth && !hasPushScope {
		return nil, fmt.Errorf("your OAuth session does not authorize HTTPS pushes; run \"tg auth login\" again")
	}
	atClient, did, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	token, err := atClient.GetServiceAuth(ctx, serviceDID(host), "org.tangled.git.receivePack")
	if err != nil {
		return nil, fmt.Errorf("mint push token for %q: %w", host, err)
	}
	return &GitCredentialResult{
		Token:                token,
		Handle:               s.resolveAuthor(ctx, did).Handle,
		MatchesRequestedHost: true,
	}, nil
}
