package atproto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestActivateOAuthSessionClearsPasswordAuth(t *testing.T) {
	dir := t.TempDir()
	m, err := NewAuthManager("http://127.0.0.1/callback", dir)
	if err != nil {
		t.Fatal(err)
	}
	did := syntax.DID("did:plc:password")
	if err := m.savePasswordSession(&atclient.PasswordSessionData{AccountDID: did, AccessToken: "access", RefreshToken: "refresh", Host: "https://pds.example"}); err != nil {
		t.Fatal(err)
	}
	m.state = authState{CurrentDID: did.String(), Method: "password"}

	if err := m.activateOAuthSession("did:plc:oauth", "oauth-session"); err != nil {
		t.Fatal(err)
	}
	if m.state.Method != "oauth" || m.state.CurrentDID != "did:plc:oauth" || m.state.CurrentSession != "oauth-session" {
		t.Fatalf("unexpected OAuth state: %+v", m.state)
	}
	if m.passwordSession != nil {
		t.Fatal("password session remained loaded")
	}
	if _, err := os.Stat(m.passwordPath); !os.IsNotExist(err) {
		t.Fatalf("password session file was not removed: %v", err)
	}
}

func TestNewAuthManagerRecoversFromMissingPasswordSession(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(authState{CurrentDID: "did:plc:stale", Method: "password"})
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := NewAuthManager("http://127.0.0.1/callback", dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.IsAuthenticated() {
		t.Fatal("stale password state should be cleared")
	}
}

func TestNewAuthManagerRecoversFromCorruptPasswordSession(t *testing.T) {
	dir := t.TempDir()
	state, _ := json.Marshal(authState{CurrentDID: "did:plc:stale", Method: "password"})
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), state, 0o600); err != nil {
		t.Fatal(err)
	}
	passwordPath := filepath.Join(dir, "password-session.json")
	if err := os.WriteFile(passwordPath, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := NewAuthManager("http://127.0.0.1/callback", dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.IsAuthenticated() {
		t.Fatal("corrupt password state should be cleared")
	}
	if _, err := os.Stat(passwordPath); !os.IsNotExist(err) {
		t.Fatalf("corrupt password session was not removed: %v", err)
	}
}

func TestNewAuthManagerRecoversFromIncompletePasswordSession(t *testing.T) {
	dir := t.TempDir()
	state, _ := json.Marshal(authState{CurrentDID: "did:plc:stale", Method: "password"})
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), state, 0o600); err != nil {
		t.Fatal(err)
	}
	session, _ := json.Marshal(atclient.PasswordSessionData{AccountDID: syntax.DID("did:plc:stale")})
	passwordPath := filepath.Join(dir, "password-session.json")
	if err := os.WriteFile(passwordPath, session, 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := NewAuthManager("http://127.0.0.1/callback", dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.IsAuthenticated() {
		t.Fatal("incomplete password state should be cleared")
	}
}

func TestPasswordLogoutRevokesAndRemovesSession(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.deleteSession" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dir := t.TempDir()
	did := syntax.DID("did:plc:test")
	m := &AuthManager{statePath: filepath.Join(dir, "auth.json"), passwordPath: filepath.Join(dir, "password-session.json"), state: authState{CurrentDID: did.String(), Method: "password"}}
	if err := m.savePasswordSession(&atclient.PasswordSessionData{AccountDID: did, AccessToken: "access", RefreshToken: "refresh", Host: server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := m.Logout(context.Background()); err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer refresh" {
		t.Fatalf("authorization = %q", authorization)
	}
	if m.IsAuthenticated() {
		t.Fatal("manager remained authenticated")
	}
	if _, err := os.Stat(m.passwordPath); !os.IsNotExist(err) {
		t.Fatalf("password session file was not removed: %v", err)
	}
}
