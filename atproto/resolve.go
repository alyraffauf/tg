package atproto

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Resolver wraps an identity.Directory to provide typed handle and
// DID resolution with a sane default timeout.
type Resolver struct {
	Directory identity.Directory
}

func (r *Resolver) ResolveHandle(ctx context.Context, rawHandle string) (*identity.Identity, error) {
	handle, err := syntax.ParseHandle(rawHandle)
	if err != nil {
		return nil, fmt.Errorf("parse handle %q: %w", rawHandle, err)
	}

	resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	ident, err := r.Directory.LookupHandle(resolveCtx, handle)
	if err != nil {
		return nil, fmt.Errorf("lookup handle %q: %w", rawHandle, err)
	}

	return ident, nil
}

func (r *Resolver) ResolveDID(ctx context.Context, didStr string) (*identity.Identity, error) {
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return nil, fmt.Errorf("parse DID %q: %w", didStr, err)
	}

	resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	ident, err := r.Directory.LookupDID(resolveCtx, did)
	if err != nil {
		return nil, fmt.Errorf("lookup DID %q: %w", didStr, err)
	}

	return ident, nil
}

func (r *Resolver) ResolvePDS(ctx context.Context, didStr string) (string, error) {
	ident, err := r.ResolveDID(ctx, didStr)
	if err != nil {
		return "", err
	}

	for _, svc := range ident.Services {
		if svc.Type == "AtprotoPersonalDataServer" {
			return svc.URL, nil
		}
	}

	return "", fmt.Errorf("no AtprotoPersonalDataServer service found for %q", didStr)
}
