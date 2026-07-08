package atproto

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const serviceAuthTTL = 60 * time.Second

// GetServiceAuth mints a short-lived service-auth JWT scoped to one lexicon
// method on one audience (e.g. a knot's did:web). Present it to that audience
// as a Bearer token.
func GetServiceAuth(ctx context.Context, pds *atclient.APIClient, audience, lexiconMethod string) (string, error) {
	if pds == nil {
		return "", fmt.Errorf("PDS client is required")
	}
	var out struct {
		Token string `json:"token"`
	}
	params := map[string]any{
		"aud": audience,
		"exp": time.Now().Add(serviceAuthTTL).Unix(),
		"lxm": lexiconMethod,
	}
	if err := pds.Get(ctx, syntax.NSID("com.atproto.server.getServiceAuth"), params, &out); err != nil {
		return "", fmt.Errorf("get service auth for %q: %w", audience, err)
	}
	if out.Token == "" {
		return "", fmt.Errorf("PDS returned an empty service auth token for %q", audience)
	}
	return out.Token, nil
}
