package atproto

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

	got, err := store.GetPasswordSession(ctx)
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
	_, err := store.GetPasswordSession(context.Background())
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
	if _, err := store.GetPasswordSession(ctx); !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("password session should have been removed, got: %v", err)
	}
}
