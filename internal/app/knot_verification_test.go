package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testKnotOwnerDID = "did:plc:owner"

func TestKnotOwnerVerification(t *testing.T) {
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
		{name: "oversized response", status: http.StatusOK, body: strings.Repeat("x", knotOwnerResponseMax+1), wantErr: "response exceeds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := knotOwnerRoundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != "https://knot.example/xrpc/sh.tangled.owner" {
					t.Fatalf("owner request URL = %q", request.URL)
				}
				if request.Method != http.MethodGet {
					t.Fatalf("owner request method = %q, want GET", request.Method)
				}
				if authorization := request.Header.Get("Authorization"); authorization != "" {
					t.Fatalf("owner request Authorization = %q, want none", authorization)
				}
				return &http.Response{
					StatusCode: tt.status,
					Status:     http.StatusText(tt.status),
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					Header:     make(http.Header),
				}, nil
			})
			verifier := &httpKnotOwnershipVerifier{client: &http.Client{Transport: transport}}

			err := verifier.Verify(context.Background(), "knot.example", testKnotOwnerDID)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("Verify() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestKnotOwnerVerificationRejectsRedirects(t *testing.T) {
	verifier := newHTTPKnotOwnershipVerifier()
	tests := []string{
		"https://other.example/xrpc/sh.tangled.owner",
		"http://knot.example/xrpc/sh.tangled.owner",
	}
	for _, target := range tests {
		request, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatalf("create redirect request: %v", err)
		}
		if err := verifier.client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
			t.Errorf("CheckRedirect(%q) error = %v, want ErrUseLastResponse", target, err)
		}
	}
}

func TestKnotOwnerVerificationAllowsTrustedLocalEndpoint(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/sh.tangled.owner" {
			t.Errorf("owner request path = %q", request.URL.Path)
		}
		if _, err := io.WriteString(response, `{"owner":"did:plc:owner"}`); err != nil {
			t.Errorf("write owner response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	verifier := &httpKnotOwnershipVerifier{client: server.Client()}

	err := verifier.Verify(context.Background(), strings.TrimPrefix(server.URL, "https://"), testKnotOwnerDID)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

type knotOwnerRoundTripFunc func(*http.Request) (*http.Response, error)

func (f knotOwnerRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
