package app

import (
	"context"
	"fmt"
	"strings"
)

// GitPushToken returns credentials only when requestedHost is the current
// repository's recorded Knot.
func (s *Service) GitPushToken(ctx context.Context, requestedHost string) (*GitCredentialResult, error) {
	_, repo, err := s.repoFromCWD(ctx)
	if err != nil {
		return nil, err
	}
	host, err := parseKnotHostname(repo.Value.Knot)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(requestedHost), host) {
		return &GitCredentialResult{}, nil
	}
	hasPushScope, isOAuth, err := s.sessions.OAuthSessionHasScope(ctx, "rpc:sh.tangled.repo.push?aud=*")
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
	token, err := atClient.GetServiceAuth(ctx, "did:web:"+host, "sh.tangled.repo.push")
	if err != nil {
		return nil, fmt.Errorf("mint push token for %q: %w", host, err)
	}
	return &GitCredentialResult{
		Token:                token,
		Handle:               s.resolveAuthor(ctx, did).Handle,
		MatchesRequestedHost: true,
	}, nil
}
