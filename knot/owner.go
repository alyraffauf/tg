package knot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const ownerResponseMax = 64 << 10

// OwnershipVerifier confirms that a Knot is owned by the expected DID.
type OwnershipVerifier struct {
	client *http.Client
}

// NewOwnershipVerifier returns a verifier using client for requests.
func NewOwnershipVerifier(client *http.Client) *OwnershipVerifier {
	if client == nil {
		client = &http.Client{}
	}
	clientCopy := *client
	if clientCopy.Timeout == 0 {
		clientCopy.Timeout = 10 * time.Second
	}
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &OwnershipVerifier{client: &clientCopy}
}

// Verify checks the public sh.tangled.owner endpoint on host.
func (v *OwnershipVerifier) Verify(ctx context.Context, host, expectedOwnerDID string) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+"/xrpc/sh.tangled.owner", nil)
	if err != nil {
		return fmt.Errorf("create owner request: %w", err)
	}
	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetch Knot owner: %w", err)
	}
	defer func() { err = errors.Join(err, response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch Knot owner: unexpected HTTP status %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, ownerResponseMax+1))
	if err != nil {
		return fmt.Errorf("read Knot owner: %w", err)
	}
	if len(body) > ownerResponseMax {
		return fmt.Errorf("read Knot owner: response exceeds %d bytes", ownerResponseMax)
	}
	var output struct {
		Owner string `json:"owner"`
	}
	if err := json.Unmarshal(body, &output); err != nil {
		return fmt.Errorf("decode Knot owner: %w", err)
	}
	if output.Owner == "" {
		return fmt.Errorf("decode Knot owner: response omitted owner DID")
	}
	if output.Owner != expectedOwnerDID {
		return fmt.Errorf("knot owner mismatch: got %q, want %q", output.Owner, expectedOwnerDID)
	}
	return nil
}
