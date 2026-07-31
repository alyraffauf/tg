package atproto

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const SERVICE_AUTH_TTL = 60 * time.Second

// GetServiceAuth mints a short-lived service-auth JWT scoped to one lexicon
// method on one audience (e.g. a knot's did:web). Present it to that audience
// as a Bearer token.
func (a *ATProto) GetServiceAuth(ctx context.Context, audience, lexiconMethod string) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	params := map[string]any{
		"aud": audience,
		"exp": time.Now().Add(SERVICE_AUTH_TTL).Unix(),
		"lxm": lexiconMethod,
	}
	if err := a.Client.Get(ctx, syntax.NSID("com.atproto.server.getServiceAuth"), params, &out); err != nil {
		return "", fmt.Errorf("get service auth for %q: %w", audience, err)
	}
	if out.Token == "" {
		return "", fmt.Errorf("PDS returned an empty service auth token for %q", audience)
	}
	return out.Token, nil
}
