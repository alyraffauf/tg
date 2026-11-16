package atproto

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

func TestInsecureOAuthLoginResumeRefreshAndLogoutWithoutKeyring(t *testing.T) {
	ctx := context.Background()
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:45678/callback")
	makeKeyringUnavailable(t, manager)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	directory := identity.NewMockDirectory()
	directory.Insert(identity.Identity{
		DID: did, Handle: syntax.Handle("alice.example"),
		Services: map[string]identity.ServiceEndpoint{"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://pds.example"}},
	})
	manager.app.Dir = directory
	var state string
	var originalClientID string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		status := http.StatusOK
		var body string
		switch request.URL.Path {
		case "/.well-known/oauth-protected-resource":
			body = `{"resource":"https://pds.example","authorization_servers":["https://auth.example"]}`
		case "/.well-known/oauth-authorization-server":
			body = `{
				"issuer":"https://auth.example",
				"authorization_endpoint":"https://auth.example/authorize",
				"token_endpoint":"https://auth.example/token",
				"revocation_endpoint":"https://auth.example/revoke",
				"pushed_authorization_request_endpoint":"https://auth.example/par",
				"response_types_supported":["code"],
				"grant_types_supported":["authorization_code","refresh_token"],
				"code_challenge_methods_supported":["S256"],
				"token_endpoint_auth_methods_supported":["none","private_key_jwt"],
				"token_endpoint_auth_signing_alg_values_supported":["ES256"],
				"dpop_signing_alg_values_supported":["ES256"],
				"scopes_supported":["atproto"],
				"authorization_response_iss_parameter_supported":true,
				"require_pushed_authorization_requests":true,
				"client_id_metadata_document_supported":true
			}`
		case "/par":
			if err := request.ParseForm(); err != nil {
				t.Fatal(err)
			}
			state = request.Form.Get("state")
			originalClientID = request.Form.Get("client_id")
			status = http.StatusCreated
			body = `{"request_uri":"urn:ietf:params:oauth:request_uri:test","expires_in":300}`
		case "/token":
			if err := request.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if request.Form.Get("client_id") != originalClientID {
				t.Fatal("token request lost original OAuth client identity")
			}
			access, refresh := "initial-access", "initial-refresh"
			if request.Form.Get("grant_type") == "refresh_token" {
				if request.Form.Get("refresh_token") != "initial-refresh" {
					t.Fatal("refresh token not restored")
				}
				access, refresh = "new-access", "new-refresh"
			}
			data, err := json.Marshal(map[string]string{
				"access_token": access, "refresh_token": refresh, "token_type": "DPoP",
				"sub": did.String(), "scope": strings.Join(DefaultScopes, " "),
			})
			if err != nil {
				t.Fatal(err)
			}
			body = string(data)
		case "/xrpc/com.atproto.server.getSession":
			if request.Header.Get("Authorization") != "DPoP new-access" {
				t.Fatal("API client did not use refreshed token")
			}
			body = "{}"
		case "/revoke":
			status = http.StatusInternalServerError
			body = "{}"
		default:
			t.Fatalf("unexpected request: %s", request.URL)
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	manager.app.Client = client
	manager.app.Resolver.Client = client
	loginURL, err := manager.StartLogin(ctx, "alice.example", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(loginURL, "https://auth.example/authorize?") || state == "" {
		t.Fatal("missing authorization URL or state")
	}
	if err := manager.FinishLogin(ctx, url.Values{"state": {state}, "iss": {"https://auth.example"}, "code": {"approved"}}); err != nil {
		t.Fatal(err)
	}
	manager.CancelLogin()
	if _, err := manager.insecureStore.GetAuthRequestInfo(ctx, state); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("pending request was not removed: %v", err)
	}
	file, err := os.Stat(manager.insecureStore.path)
	if err != nil || file.Mode().Perm() != 0o600 {
		t.Fatalf("credential permissions: %v, %v", file, err)
	}

	restored := newAuthManagerForTest("", manager.store)
	restored.insecureStore = &insecureFileStore{path: manager.insecureStore.path}
	restored.app.Client = client
	accounts, active, err := restored.Accounts()
	if err != nil || len(accounts) != 1 || active != did.String() || accounts[0].Method != AuthMethodOAuth || accounts[0].Handle != "alice.example" {
		t.Fatalf("accounts=%v active=%s err=%v", accounts, active, err)
	}
	session, err := restored.CurrentSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.RefreshTokens(ctx); err != nil {
		t.Fatal(err)
	}
	saved, err := restored.insecureStore.GetSession(ctx, did, "")
	if err != nil || saved.AccessToken != "new-access" || saved.RefreshToken != "new-refresh" {
		t.Fatalf("refreshed credentials not persisted: %v", err)
	}
	record, err := restored.insecureStore.load()
	if err != nil || record.ClientID != originalClientID || !record.Active || record.Handle != "alice.example" {
		t.Fatalf("refresh lost account metadata: %v", err)
	}
	status, gotDID, err := restored.SessionStatus(ctx)
	if err != nil || status != SessionStatusActive || gotDID != did {
		t.Fatalf("status=%s did=%s err=%v", status, gotDID, err)
	}
	if err := restored.Logout(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(restored.insecureStore.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("credentials remain after logout: %v", err)
	}
	if len(manager.store.backend.(*fakeKeyring).secrets) != 0 {
		t.Fatal("insecure OAuth wrote to the keyring")
	}
}

func TestInsecureOAuthScopeMismatchRemovesFileSession(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:45678/callback")
	makeKeyringUnavailable(t, manager)
	session := fullyPopulatedSession(t, mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff"))
	session.Scopes = []string{"atproto"}
	if err := manager.insecureStore.SaveSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	_, err := manager.CurrentSession(context.Background())
	if !errors.Is(err, ErrOAuthScopeMismatch) {
		t.Fatalf("scope mismatch: %v", err)
	}
	if _, err := os.Stat(manager.insecureStore.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid session not removed: %v", err)
	}
}

func TestInsecureOAuthPendingRequestStateAndCancellation(t *testing.T) {
	manager := newAuthManagerWithFileStore(t, "http://127.0.0.1:45678/callback")
	ctx := context.Background()
	request := oauth.AuthRequestData{State: "pending-state", PKCEVerifier: "verifier"}
	if err := manager.insecureStore.SaveAuthRequestInfo(ctx, request); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.insecureStore.GetAuthRequestInfo(ctx, "other-state"); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("wrong state accepted: %v", err)
	}
	if err := manager.insecureStore.DeleteAuthRequestInfo(ctx, "other-state"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.insecureStore.GetAuthRequestInfo(ctx, request.State); err != nil {
		t.Fatalf("unrelated state removed pending request: %v", err)
	}
	manager.CancelLogin()
	if _, err := manager.insecureStore.GetAuthRequestInfo(ctx, request.State); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("canceled request remains: %v", err)
	}
	if _, err := os.Stat(manager.insecureStore.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("pending request was written to disk: %v", err)
	}
}
