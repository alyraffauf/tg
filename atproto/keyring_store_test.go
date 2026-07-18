package atproto

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

type fakeKeyring struct {
	secrets   map[string]string
	deleteErr error
	getErr    error
}

func newFakeKeyring() *fakeKeyring {
	return &fakeKeyring{secrets: make(map[string]string)}
}

func backendKey(service, user string) string { return service + "\x00" + user }

func (f *fakeKeyring) Get(service, user string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	if v, ok := f.secrets[backendKey(service, user)]; ok {
		return v, nil
	}
	return "", keyring.ErrNotFound
}

func (f *fakeKeyring) Set(service, user, password string) error {
	f.secrets[backendKey(service, user)] = password
	return nil
}

func (f *fakeKeyring) Delete(service, user string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.secrets, backendKey(service, user))
	return nil
}

func newAuthManagerForTest(callbackURL string, store *KeyringStore) *AuthManager {
	config := oauth.NewLocalhostConfig(callbackURL, DefaultScopes)
	config.UserAgent = "tg"
	return &AuthManager{
		app:   oauth.NewClientApp(&config, store),
		store: store,
	}
}

func testKeyringStore(backend secretBackend) *KeyringStore {
	return &KeyringStore{backend: backend, service: keyringService}
}

func mustDID(t *testing.T, raw string) syntax.DID {
	t.Helper()
	did, err := syntax.ParseDID(raw)
	if err != nil {
		t.Fatalf("parse DID: %v", err)
	}
	return did
}

func sampleSession(did syntax.DID) oauth.ClientSessionData {
	return oauth.ClientSessionData{
		AccountDID:   did,
		SessionID:    "session-1",
		HostURL:      "https://example.com",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}
}

func mustGenerateDpopKey(t *testing.T) string {
	t.Helper()
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatalf("generate DPoP key: %v", err)
	}
	return key.Multibase()
}

func TestKeyringStore_SaveAndGetSession(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	did := mustDID(t, "did:plc:abcdefghijklmnopqrstuvwxyz")

	if err := store.SaveSession(ctx, sampleSession(did)); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	got, err := store.GetSession(ctx, did, "")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.AccessToken != "access-token" {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, "access-token")
	}
}

func TestKeyringStore_GetSessionNotFound(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	did := mustDID(t, "did:plc:abcdefghijklmnopqrstuvwxyz")

	_, err := store.GetSession(context.Background(), did, "")
	if err == nil {
		t.Fatal("expected error for missing session, got nil")
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("error does not wrap keyring.ErrNotFound: %v", err)
	}
}

func TestKeyringStore_DeleteSession(t *testing.T) {
	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	ctx := context.Background()
	did := mustDID(t, "did:plc:abcdefghijklmnopqrstuvwxyz")

	if err := store.SaveSession(ctx, sampleSession(did)); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if err := store.DeleteSession(ctx, did, ""); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if _, err := store.GetSession(ctx, did, ""); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("session still present after delete: %v", err)
	}

	if err := store.DeleteSession(ctx, did, ""); err != nil {
		t.Fatalf("deleting missing session errored: %v", err)
	}
}

func TestKeyringStore_SaveOverwritesPrevious(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	first := mustDID(t, "did:plc:firstfirstfirstfirstfirst")
	second := mustDID(t, "did:plc:secondsecondsecondsecond")

	if err := store.SaveSession(ctx, sampleSession(first)); err != nil {
		t.Fatalf("SaveSession first: %v", err)
	}
	if err := store.SaveSession(ctx, sampleSession(second)); err != nil {
		t.Fatalf("SaveSession second: %v", err)
	}

	got, err := store.GetSession(ctx, first, "")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.AccountDID != second {
		t.Errorf("AccountDID = %q, want %q (second DID)", got.AccountDID, second)
	}
}

func TestKeyringStore_AuthRequestRoundTrip(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	info := oauth.AuthRequestData{State: "state-1", PKCEVerifier: "verifier"}

	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("SaveAuthRequestInfo: %v", err)
	}
	got, err := store.GetAuthRequestInfo(ctx, "state-1")
	if err != nil {
		t.Fatalf("GetAuthRequestInfo: %v", err)
	}
	if got.PKCEVerifier != "verifier" {
		t.Errorf("PKCEVerifier = %q, want %q", got.PKCEVerifier, "verifier")
	}
	if err := store.DeleteAuthRequestInfo(ctx, "state-1"); err != nil {
		t.Fatalf("DeleteAuthRequestInfo: %v", err)
	}
}

func TestKeyringStore_DeleteSessionPropagatesError(t *testing.T) {
	backend := newFakeKeyring()
	backend.deleteErr = errors.New("keyring daemon unavailable")
	store := testKeyringStore(backend)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	if err := store.SaveSession(ctx, sampleSession(did)); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	err := store.DeleteSession(ctx, did, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "keyring daemon unavailable") {
		t.Errorf("error = %q, want substring %q", err, "keyring daemon unavailable")
	}
}

func TestAuthManager_CurrentSessionErrNotFound(t *testing.T) {
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", testKeyringStore(newFakeKeyring()))
	_, err := manager.CurrentSession(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestAuthManager_CurrentSessionPropagatesGetError(t *testing.T) {
	backend := newFakeKeyring()
	backend.getErr = errors.New("dbus connection refused")
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", testKeyringStore(backend))
	_, err := manager.CurrentSession(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrNotAuthenticated) {
		t.Error("keyring access error should not be classified as unauthenticated")
	}
	if !strings.Contains(err.Error(), "dbus connection refused") {
		t.Errorf("error = %q, want substring %q", err, "dbus connection refused")
	}
}

func TestAuthManager_LogoutWithoutSession(t *testing.T) {
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", testKeyringStore(newFakeKeyring()))
	err := manager.Logout(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("expected ErrNotAuthenticated for logout without session, got %v", err)
	}
}

func TestAuthManager_LogoutDeletionErrorPropagates(t *testing.T) {
	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	session := sampleSession(did)
	session.DPoPPrivateKeyMultibase = mustGenerateDpopKey(t)
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	backend.deleteErr = errors.New("keyring daemon unavailable")
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	err := manager.Logout(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "keyring daemon unavailable") {
		t.Errorf("error = %q, want substring %q", err, "keyring daemon unavailable")
	}
}

// fullyPopulatedSession returns a ClientSessionData with every field set to a
// distinct, non-zero value, so a round-trip test catches any field that gets
// dropped or mis-serialized.
func fullyPopulatedSession(t *testing.T, did syntax.DID) oauth.ClientSessionData {
	t.Helper()
	return oauth.ClientSessionData{
		AccountDID:                   did,
		SessionID:                    "session-1",
		HostURL:                      "https://pds.example.com",
		AuthServerURL:                "https://auth.example.com",
		AuthServerTokenEndpoint:      "https://auth.example.com/token",
		AuthServerRevocationEndpoint: "https://auth.example.com/revoke",
		Scopes:                       []string{"atproto", "repo:sh.tangled.repo"},
		AccessToken:                  "access-token-123",
		RefreshToken:                 "refresh-token-456",
		DPoPAuthServerNonce:          "authserver-nonce",
		DPoPHostNonce:                "host-nonce",
		DPoPPrivateKeyMultibase:      mustGenerateDpopKey(t),
	}
}

// TestKeyringStore_SessionRoundTrip_FullyPopulated verifies that every field
// of ClientSessionData survives a save→load cycle, including the DPoP private
// key, scopes, and the omitempty revocation endpoint.
func TestKeyringStore_SessionRoundTrip_FullyPopulated(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	want := fullyPopulatedSession(t, did)

	if err := store.SaveSession(ctx, want); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	got, err := store.GetSession(ctx, did, "")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", *got, want)
	}
}

// TestKeyringStore_SessionRoundTrip_EmptyScopesAndRevocation verifies the
// omitempty/empty-slice edge cases (empty scopes slice, empty revocation
// endpoint) round-trip without losing the distinction that matters.
func TestKeyringStore_SessionRoundTrip_EmptyScopesAndRevocation(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	want := fullyPopulatedSession(t, did)
	want.AuthServerRevocationEndpoint = ""
	want.Scopes = []string{}

	if err := store.SaveSession(ctx, want); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	got, err := store.GetSession(ctx, did, "")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.AuthServerRevocationEndpoint != "" {
		t.Errorf("revocation endpoint = %q, want empty", got.AuthServerRevocationEndpoint)
	}
	if len(got.Scopes) != 0 {
		t.Errorf("scopes = %v, want empty", got.Scopes)
	}
}

// TestKeyringStore_AuthRequestRoundTrip_FullyPopulated verifies every field of
// AuthRequestData round-trips, including the *syntax.DID pointer in both the
// non-nil and nil cases.
func TestKeyringStore_AuthRequestRoundTrip_FullyPopulated(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	want := oauth.AuthRequestData{
		State:                        "state-1",
		AuthServerURL:                "https://auth.example.com",
		AccountDID:                   &did,
		Scopes:                       []string{"atproto", "repo:sh.tangled.repo"},
		RequestURI:                   "urn:ietf:params:oauth:request_uri:abc",
		AuthServerTokenEndpoint:      "https://auth.example.com/token",
		AuthServerRevocationEndpoint: "https://auth.example.com/revoke",
		PKCEVerifier:                 "verifier-123",
		DPoPAuthServerNonce:          "nonce-123",
		DPoPPrivateKeyMultibase:      mustGenerateDpopKey(t),
	}
	if err := store.SaveAuthRequestInfo(ctx, want); err != nil {
		t.Fatalf("SaveAuthRequestInfo: %v", err)
	}
	got, err := store.GetAuthRequestInfo(ctx, "state-1")
	if err != nil {
		t.Fatalf("GetAuthRequestInfo: %v", err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", *got, want)
	}

	// nil AccountDID must round-trip as nil (omitempty drops it).
	wantNil := want
	wantNil.State = "state-2"
	wantNil.AccountDID = nil
	if err := store.SaveAuthRequestInfo(ctx, wantNil); err != nil {
		t.Fatalf("SaveAuthRequestInfo (nil DID): %v", err)
	}
	gotNil, err := store.GetAuthRequestInfo(ctx, "state-2")
	if err != nil {
		t.Fatalf("GetAuthRequestInfo (nil DID): %v", err)
	}
	if gotNil.AccountDID != nil {
		t.Errorf("AccountDID = %v, want nil", gotNil.AccountDID)
	}
}

// TestKeyringStore_GetSessionIgnoresDID verifies the singleton contract the
// codebase relies on: the did and sessionID arguments are ignored, and the
// stored session is returned regardless of what is requested. This nails down
// the design so a future multi-session refactor is caught.
func TestKeyringStore_GetSessionIgnoresDID(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	savedDID := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	if err := store.SaveSession(ctx, sampleSession(savedDID)); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	otherDID := mustDID(t, "did:plc:zzzzzzzzzzzzzzzzzzzzzzzz")
	got, err := store.GetSession(ctx, otherDID, "nonexistent-session")
	if err != nil {
		t.Fatalf("GetSession with different DID: %v", err)
	}
	if got.AccountDID != savedDID {
		t.Errorf("AccountDID = %q, want %q (singleton ignores requested DID)", got.AccountDID, savedDID)
	}
}

// TestKeyringStore_GetSessionMalformedJSON verifies that a corrupt keyring
// entry (e.g. written by an older build or another program) produces a
// clear, non-ErrNotFound error rather than being mistaken for "no session".
func TestKeyringStore_GetSessionMalformedJSON(t *testing.T) {
	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	// Seed a corrupt entry directly under the session key.
	backend.secrets[backendKey(keyringService, currentSessionKey)] = "not-json{"

	_, err := store.GetSession(context.Background(), syntax.DID(""), "")
	if err == nil {
		t.Fatal("expected error for corrupt session, got nil")
	}
	if errors.Is(err, keyring.ErrNotFound) {
		t.Error("corrupt entry should not be classified as not-found")
	}
	if !strings.Contains(err.Error(), "decode secret") {
		t.Errorf("error = %q, want substring %q", err, "decode secret")
	}
}

// TestKeyringStore_ConcurrentAccess exercises concurrent reads/writes against
// a single store under the race detector. It verifies the process-local mutex
// keeps operations from tearing (the fake keyring's map would otherwise race).
func TestKeyringStore_ConcurrentAccess(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	session := fullyPopulatedSession(t, did)

	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				if err := store.SaveSession(ctx, session); err != nil {
					t.Errorf("SaveSession: %v", err)
					return
				}
				if _, err := store.GetSession(ctx, did, ""); err != nil {
					t.Errorf("GetSession: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

// TestAuthManager_PersistSessionCallbackPersistsRefresh verifies that the
// PersistSessionCallback indigo wires during ResumeSession actually writes
// rotated tokens back to the keyring. This is the property that lets a token
// refresh survive a process restart.
func TestAuthManager_PersistSessionCallbackPersistsRefresh(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	session := fullyPopulatedSession(t, did)
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	sess, err := manager.CurrentSession(ctx)
	if err != nil {
		t.Fatalf("CurrentSession: %v", err)
	}
	if sess.PersistSessionCallback == nil {
		t.Fatal("PersistSessionCallback not wired by ResumeSession")
	}

	// Simulate a token refresh: rotate both tokens and persist.
	sess.Data.AccessToken = "rotated-access-token"
	sess.Data.RefreshToken = "rotated-refresh-token"
	sess.PersistSessionCallback(ctx, sess.Data)

	reloaded, err := store.GetSession(ctx, did, "")
	if err != nil {
		t.Fatalf("GetSession after refresh: %v", err)
	}
	if reloaded.AccessToken != "rotated-access-token" {
		t.Errorf("AccessToken = %q, want %q", reloaded.AccessToken, "rotated-access-token")
	}
	if reloaded.RefreshToken != "rotated-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", reloaded.RefreshToken, "rotated-refresh-token")
	}
}

// TestAuthManager_LogoutClearsCorruptSession verifies that Logout recovers
// from a corrupt session entry (invalid DPoP key) by force-clearing it, so the
// user can re-login instead of being permanently locked out.
func TestAuthManager_LogoutClearsCorruptSession(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	ctx := context.Background()
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")

	// Save a session with an invalid (empty) DPoP private key. ResumeSession
	// will fail to parse it, so indigo's Logout never reaches DeleteSession.
	corrupt := sampleSession(did)
	corrupt.DPoPPrivateKeyMultibase = ""
	if err := store.SaveSession(ctx, corrupt); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	if err := manager.Logout(ctx); err != nil {
		t.Fatalf("Logout should force-clear a corrupt session, got: %v", err)
	}
	if _, err := store.GetSession(ctx, did, ""); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("corrupt session should have been cleared, got: %v", err)
	}
}

// TestAuthManager_CancelLoginDeletesPendingRequest verifies that an abandoned
// login (StartLogin without FinishLogin) leaves no auth-request entry behind
// once CancelLogin is called.
func TestAuthManager_CancelLoginDeletesPendingRequest(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	ctx := context.Background()
	info := oauth.AuthRequestData{
		State:                   "pending-state",
		PKCEVerifier:            "verifier",
		DPoPPrivateKeyMultibase: mustGenerateDpopKey(t),
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("SaveAuthRequestInfo: %v", err)
	}

	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	manager.CancelLogin()

	if _, err := store.GetAuthRequestInfo(ctx, "pending-state"); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("pending auth request should have been cleared, got: %v", err)
	}

	// CancelLogin after the request is already gone is a no-op.
	manager.CancelLogin()
}
