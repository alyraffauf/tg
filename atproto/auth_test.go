package atproto

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

func samplePasswordSession(did syntax.DID, host string) atclient.PasswordSessionData {
	return atclient.PasswordSessionData{
		AccountDID:   did,
		AccessToken:  "access",
		RefreshToken: "refresh",
		Host:         host,
	}
}

func TestPasswordSessionRoundTrip(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	want := samplePasswordSession(did, "https://pds.example")
	if err := store.SavePasswordSession(ctx, want); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	got, err := store.GetPasswordSession(ctx, did)
	if err != nil {
		t.Fatalf("GetPasswordSession: %v", err)
	}
	if got.AccountDID != want.AccountDID {
		t.Errorf("AccountDID = %q, want %q", got.AccountDID, want.AccountDID)
	}
	if got.AccessToken != want.AccessToken {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, want.AccessToken)
	}
	if got.RefreshToken != want.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, want.RefreshToken)
	}
	if got.Host != want.Host {
		t.Errorf("Host = %q, want %q", got.Host, want.Host)
	}
}

func TestPasswordSessionNotFound(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	_, err := store.GetPasswordSession(context.Background(), "")
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("GetPasswordSession = %v, want keyring.ErrNotFound", err)
	}
}

func TestCurrentDIDFallsBackToPassword(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, "https://pds.example")); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	got, err := manager.CurrentDID(ctx)
	if err != nil {
		t.Fatalf("CurrentDID: %v", err)
	}
	if got != did {
		t.Errorf("CurrentDID = %q, want %q", got, did)
	}
}

func TestCurrentDIDNotAuthenticated(t *testing.T) {
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", testKeyringStore(newFakeKeyring()))
	_, err := manager.CurrentDID(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("CurrentDID = %v, want ErrNotAuthenticated", err)
	}
}

func TestCurrentDIDRejectsStalePasswordIndex(t *testing.T) {
	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	if err := store.SavePasswordSession(context.Background(), samplePasswordSession(did, "https://pds.example")); err != nil {
		t.Fatal(err)
	}
	delete(backend.secrets, backendKey(keyringService, passwordKey(did.String())))
	if _, err := manager.CurrentDID(context.Background()); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("CurrentDID = %v, want ErrNotAuthenticated", err)
	}
}

func TestCurrentDIDRejectsStaleOAuthIndex(t *testing.T) {
	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	if err := store.SaveSession(context.Background(), sampleSession(did)); err != nil {
		t.Fatal(err)
	}
	delete(backend.secrets, backendKey(keyringService, sessionKey(did.String())))
	if _, err := manager.CurrentDID(context.Background()); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("CurrentDID = %v, want ErrNotAuthenticated", err)
	}
}

func TestAPIClientReturnsPasswordClient(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, "https://pds.example")); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	client, gotDID, err := manager.APIClient(ctx)
	if err != nil {
		t.Fatalf("APIClient: %v", err)
	}
	if gotDID != did {
		t.Errorf("DID = %q, want %q", gotDID, did)
	}
	if _, ok := client.Auth.(*atclient.PasswordAuth); !ok {
		t.Errorf("Auth = %T, want *atclient.PasswordAuth", client.Auth)
	}
}

func TestAPIClientNotAuthenticated(t *testing.T) {
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", testKeyringStore(newFakeKeyring()))
	_, _, err := manager.APIClient(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("APIClient = %v, want ErrNotAuthenticated", err)
	}
}

// TestPasswordLogoutRevokesAndRemovesSession verifies that logging out of a
// password session revokes it at the PDS (using the refresh token) and then
// deletes it from the keyring.
func TestPasswordLogoutRevokesAndRemovesSession(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.deleteSession" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	if err := manager.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if authorization != "Bearer refresh" {
		t.Errorf("Authorization = %q, want %q", authorization, "Bearer refresh")
	}
	if _, err := store.GetPasswordSession(ctx, did); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("password session should have been removed, got: %v", err)
	}
}

func TestPasswordLogoutPreservesOtherAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	first := mustDID(t, "did:plc:firstfirstfirstfirstfirst")
	second := mustDID(t, "did:plc:secondsecondsecondsecond")
	if err := store.SavePasswordSession(ctx, samplePasswordSession(first, server.URL)); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAccountHandle(first.String(), "first.example"); err != nil {
		t.Fatal(err)
	}
	if err := store.SavePasswordSession(ctx, samplePasswordSession(second, server.URL)); err != nil {
		t.Fatal(err)
	}
	if err := store.SetAccountHandle(second.String(), "second.example"); err != nil {
		t.Fatal(err)
	}
	manager.SetAccount("second.example")
	if err := manager.Logout(ctx); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := store.GetPasswordSession(ctx, second); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("logged-out account remains: %v", err)
	}
	if _, err := store.GetPasswordSession(ctx, first); err != nil {
		t.Fatalf("other account was removed: %v", err)
	}
}

func TestAccountOverrideSelectsWithoutChangingDefault(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	first := mustDID(t, "did:plc:firstfirstfirstfirstfirst")
	second := mustDID(t, "did:plc:secondsecondsecondsecond")
	if err := store.SavePasswordSession(ctx, samplePasswordSession(first, "https://one.example")); err != nil {
		t.Fatal(err)
	}
	if err := store.SavePasswordSession(ctx, samplePasswordSession(second, "https://two.example")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SelectAccount(first.String()); err != nil {
		t.Fatal(err)
	}
	manager.SetAccount(second.String())
	_, did, err := manager.APIClient(ctx)
	if err != nil || did != second {
		t.Fatalf("override DID = %q, err = %v", did, err)
	}
	_, active, err := store.Accounts()
	if err != nil || active != first.String() {
		t.Fatalf("persistent active = %q, want %q (err %v)", active, first, err)
	}
}

func TestLogoutAllRevokesEveryAccount(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	first := mustDID(t, "did:plc:firstfirstfirstfirstfirst")
	second := mustDID(t, "did:plc:secondsecondsecondsecond")
	if err := store.SavePasswordSession(ctx, samplePasswordSession(first, server.URL)); err != nil {
		t.Fatal(err)
	}
	if err := store.SavePasswordSession(ctx, samplePasswordSession(second, server.URL)); err != nil {
		t.Fatal(err)
	}
	if err := manager.LogoutAll(ctx); err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	if requests != 2 {
		t.Fatalf("deleteSession requests = %d, want 2", requests)
	}
	accounts, active, err := store.Accounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 0 || active != "" {
		t.Fatalf("accounts after logout = %#v active %q", accounts, active)
	}
}

func TestPasswordLogoutClearsLocalOn401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"AuthenticationFailed","message":"Authentication failed"}`))
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	if err := manager.Logout(ctx); err != nil {
		t.Fatalf("Logout should clear local state despite 401, got: %v", err)
	}
	if _, err := store.GetPasswordSession(ctx, did); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("password session should have been cleared, got: %v", err)
	}
}

func TestPasswordLogoutClearsLocalOn500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	if err := manager.Logout(ctx); err != nil {
		t.Fatalf("Logout should clear local state despite 500, got: %v", err)
	}
	if _, err := store.GetPasswordSession(ctx, did); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("password session should have been cleared, got: %v", err)
	}
}

func TestPasswordLogoutFailsWhenLocalClearErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	backend.deleteErr = errors.New("keyring daemon unavailable")
	err := manager.Logout(ctx)
	if err == nil {
		t.Fatal("expected error when local clear fails, got nil")
	}
	if !strings.Contains(err.Error(), "clear local session") {
		t.Errorf("error = %q, want substring %q", err, "clear local session")
	}
	if !strings.Contains(err.Error(), "keyring daemon unavailable") {
		t.Errorf("error = %q, want substring %q", err, "keyring daemon unavailable")
	}
}

func TestLogoutAllClearsAllDespiteServerFailures(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"AuthenticationFailed","message":"Authentication failed"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	first := mustDID(t, "did:plc:firstfirstfirstfirstfirst")
	second := mustDID(t, "did:plc:secondsecondsecondsecond")
	if err := store.SavePasswordSession(ctx, samplePasswordSession(first, server.URL)); err != nil {
		t.Fatal(err)
	}
	if err := store.SavePasswordSession(ctx, samplePasswordSession(second, server.URL)); err != nil {
		t.Fatal(err)
	}

	if err := manager.LogoutAll(ctx); err != nil {
		t.Fatalf("LogoutAll should clear all locals despite server failures, got: %v", err)
	}
	if _, err := store.GetPasswordSession(ctx, first); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("first session should have been cleared, got: %v", err)
	}
	if _, err := store.GetPasswordSession(ctx, second); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("second session should have been cleared, got: %v", err)
	}
}

func TestOAuthLogoutClearsLocalOnServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	session := sampleSession(did)
	session.DPoPPrivateKeyMultibase = mustGenerateDpopKey(t)
	session.HostURL = server.URL
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	if err := manager.Logout(ctx); err != nil {
		t.Fatalf("Logout should clear local state despite server error, got: %v", err)
	}
	if _, err := store.GetSession(ctx, did, ""); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("OAuth session should have been cleared, got: %v", err)
	}
}

// TestPasswordLogoutClearsAccountWithMissingSession verifies that Logout
// removes the account from the index even when the session secret is already
// gone, so the user is never locked out by a stale index entry.
func TestPasswordLogoutClearsAccountWithMissingSession(t *testing.T) {
	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, "https://pds.example")); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}
	delete(backend.secrets, backendKey(keyringService, passwordKey(did.String())))

	if err := manager.Logout(ctx); err != nil {
		t.Fatalf("Logout should clear account despite missing session, got: %v", err)
	}
	accounts, _, err := store.Accounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 0 {
		t.Errorf("account should have been removed from index, got: %v", accounts)
	}
}

func TestSessionStatusNotAuthenticated(t *testing.T) {
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", testKeyringStore(newFakeKeyring()))
	_, _, err := manager.SessionStatus(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("SessionStatus = %v, want ErrNotAuthenticated", err)
	}
}

func TestSessionStatusActivePassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.getSession" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	status, gotDID, err := manager.SessionStatus(ctx)
	if err != nil {
		t.Fatalf("SessionStatus: %v", err)
	}
	if status != SessionStatusActive {
		t.Errorf("status = %q, want %q", status, SessionStatusActive)
	}
	if gotDID != did {
		t.Errorf("DID = %q, want %q", gotDID, did)
	}
}

func TestSessionStatusExpiredPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"AuthenticationFailed","message":"Authentication failed"}`))
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	status, _, err := manager.SessionStatus(ctx)
	if err != nil {
		t.Fatalf("SessionStatus: %v", err)
	}
	if status != SessionStatusExpired {
		t.Errorf("status = %q, want %q", status, SessionStatusExpired)
	}
}

func TestSessionStatusExpiredPasswordRefreshDead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/xrpc/com.atproto.server.getSession" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"ExpiredToken","message":"access token expired"}`))
			return
		}
		if r.URL.Path == "/xrpc/com.atproto.server.refreshSession" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"AuthenticationFailed","message":"Authentication failed"}`))
			return
		}
		t.Errorf("unexpected path %q", r.URL.Path)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	status, _, err := manager.SessionStatus(ctx)
	if err != nil {
		t.Fatalf("SessionStatus: %v", err)
	}
	if status != SessionStatusExpired {
		t.Errorf("status = %q, want %q", status, SessionStatusExpired)
	}
}

func TestSessionStatusUnknownPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SavePasswordSession(ctx, samplePasswordSession(did, server.URL)); err != nil {
		t.Fatalf("SavePasswordSession: %v", err)
	}

	status, _, err := manager.SessionStatus(ctx)
	if err != nil {
		t.Fatalf("SessionStatus: %v", err)
	}
	if status != SessionStatusUnknown {
		t.Errorf("status = %q, want %q", status, SessionStatusUnknown)
	}
}

func TestSessionStatusActiveOAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.getSession" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	session := sampleSession(did)
	session.DPoPPrivateKeyMultibase = mustGenerateDpopKey(t)
	session.HostURL = server.URL
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	status, gotDID, err := manager.SessionStatus(ctx)
	if err != nil {
		t.Fatalf("SessionStatus: %v", err)
	}
	if status != SessionStatusActive {
		t.Errorf("status = %q, want %q", status, SessionStatusActive)
	}
	if gotDID != did {
		t.Errorf("DID = %q, want %q", gotDID, did)
	}
}

func TestSessionStatusExpiredOAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"AuthenticationFailed","message":"Authentication failed"}`))
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	session := sampleSession(did)
	session.DPoPPrivateKeyMultibase = mustGenerateDpopKey(t)
	session.HostURL = server.URL
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	status, _, err := manager.SessionStatus(ctx)
	if err != nil {
		t.Fatalf("SessionStatus: %v", err)
	}
	if status != SessionStatusExpired {
		t.Errorf("status = %q, want %q", status, SessionStatusExpired)
	}
}

func TestSessionStatusExpiredOAuthRefreshDead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.getSession":
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/token":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	session := sampleSession(did)
	session.DPoPPrivateKeyMultibase = mustGenerateDpopKey(t)
	session.HostURL = server.URL
	session.AuthServerTokenEndpoint = server.URL + "/token"
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	status, _, err := manager.SessionStatus(ctx)
	if err != nil {
		t.Fatalf("SessionStatus: %v", err)
	}
	if status != SessionStatusExpired {
		t.Errorf("status = %q, want %q", status, SessionStatusExpired)
	}
}
