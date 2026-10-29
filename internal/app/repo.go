package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// ownerHandle resolves a DID to its handle, falling back to the raw DID.
func (s *Service) ownerHandle(ctx context.Context, did string) string {
	if ident, err := s.resolver.ResolveDID(ctx, did); err == nil {
		if handle := ident.Handle; handle.String() != "" && !handle.IsInvalidHandle() {
			return handle.String()
		}
	}
	return did
}

// parseKnotHostname normalizes a Knot hostname argument.
func parseKnotHostname(raw string) (string, error) {
	hostname, err := syntax.ParseHandle(raw)
	if err != nil {
		return "", fmt.Errorf("invalid Knot hostname %q: %w", raw, err)
	}
	return hostname.Normalize().String(), nil
}

func serviceDID(host string) string { return "did:web:" + strings.ReplaceAll(host, ":", "%3A") }
