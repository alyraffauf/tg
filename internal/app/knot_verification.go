package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	knotVerificationTimeout = 10 * time.Second
	knotOwnerResponseMax    = 64 << 10
)

type httpKnotOwnershipVerifier struct {
	client *http.Client
}

func newHTTPKnotOwnershipVerifier() *httpKnotOwnershipVerifier {
	return &httpKnotOwnershipVerifier{
		client: &http.Client{
			Timeout: knotVerificationTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (v *httpKnotOwnershipVerifier) Verify(ctx context.Context, host, expectedOwnerDID string) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+"/xrpc/sh.tangled.owner", nil)
	if err != nil {
		return fmt.Errorf("create owner request: %w", err)
	}
	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetch Knot owner: %w", err)
	}
	defer func() {
		err = errors.Join(err, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch Knot owner: unexpected HTTP status %d %s", response.StatusCode, http.StatusText(response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, knotOwnerResponseMax+1))
	if err != nil {
		return fmt.Errorf("read Knot owner: %w", err)
	}
	if len(body) > knotOwnerResponseMax {
		return fmt.Errorf("read Knot owner: response exceeds %d bytes", knotOwnerResponseMax)
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
