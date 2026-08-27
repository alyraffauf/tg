package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

const (
	knotCollection       = "sh.tangled.knot"
	maxKnotRegistrations = 10
)

type knotRegistration struct {
	LexiconTypeID string `json:"$type"`
	CreatedAt     string `json:"createdAt"`
}

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

func validateKnotRegistration(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode record: %w", err)
	}
	var registration knotRegistration
	if err := json.Unmarshal(data, &registration); err != nil {
		return fmt.Errorf("decode record: %w", err)
	}
	if registration.LexiconTypeID != knotCollection {
		return fmt.Errorf("$type must be %q", knotCollection)
	}
	if _, err := syntax.ParseDatetime(registration.CreatedAt); err != nil {
		return fmt.Errorf("invalid createdAt: %w", err)
	}
	return nil
}
