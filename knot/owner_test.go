package knot

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOwnershipVerifier(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{name: "matching owner", status: http.StatusOK, body: `{"owner":"did:plc:owner"}`},
		{name: "different owner", status: http.StatusOK, body: `{"owner":"did:plc:other"}`, wantErr: "owner mismatch"},
		{name: "missing owner", status: http.StatusOK, body: `{}`, wantErr: "omitted owner DID"},
		{name: "malformed response", status: http.StatusOK, body: `{`, wantErr: "decode Knot owner"},
		{name: "server error", status: http.StatusBadGateway, body: `{}`, wantErr: "502 Bad Gateway"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != "https://knot.example/xrpc/sh.tangled.owner" {
					t.Fatalf("owner request URL = %q", request.URL)
				}
				return &http.Response{StatusCode: tt.status, Status: http.StatusText(tt.status), Body: io.NopCloser(strings.NewReader(tt.body)), Header: make(http.Header)}, nil
			})}
			verifier := NewOwnershipVerifier(client)
			err := verifier.Verify(context.Background(), "knot.example", "did:plc:owner")
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("Verify() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestOwnershipVerifierRejectsRedirects(t *testing.T) {
	verifier := NewOwnershipVerifier(nil)
	request, _ := http.NewRequest(http.MethodGet, "https://other.example/xrpc/sh.tangled.owner", nil)
	if err := verifier.client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect() error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
