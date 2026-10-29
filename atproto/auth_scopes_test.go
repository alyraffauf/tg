package atproto

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestOAuthScopeMismatchSignsOutOnlyTheSelectedAccount(t *testing.T) {
	operations := map[string]func(*AuthManager) error{
		"current DID":     func(m *AuthManager) error { _, err := m.CurrentDID(context.Background()); return err },
		"current session": func(m *AuthManager) error { _, err := m.CurrentSession(context.Background()); return err },
		"API client":      func(m *AuthManager) error { _, _, err := m.APIClient(context.Background()); return err },
		"scope query": func(m *AuthManager) error {
			_, _, err := m.OAuthSessionHasScope(context.Background(), DefaultScopes[0])
			return err
		},
		"status": func(m *AuthManager) error { _, _, err := m.SessionStatus(context.Background()); return err },
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := testKeyringStore(newFakeKeyring())
			manager := newAuthManagerForTest("", store)
			manager.app.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Error("mismatched session made a network request")
				return nil, errors.New("unexpected request")
			})}
			did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
			session := fullyPopulatedSession(t, did)
			session.Scopes = []string{"atproto", "repo:sh.tangled.repo"}
			if err := store.SaveSession(ctx, session); err != nil {
				t.Fatal(err)
			}
			if err := store.SetAccountHandle(did.String(), "alice.example"); err != nil {
				t.Fatal(err)
			}
			other := fullyPopulatedSession(t, mustDID(t, "did:plc:bbbbccccddddeeeeffffaaaa"))
			if err := store.SaveSession(ctx, other); err != nil {
				t.Fatal(err)
			}
			manager.SetAccount("alice.example")

			err := operation(manager)
			if !errors.Is(err, ErrOAuthScopeMismatch) || !strings.Contains(err.Error(), "signed out alice.example") || !strings.Contains(err.Error(), "tg auth login alice.example") {
				t.Fatalf("error = %v", err)
			}
			// A missing-session error can make callers fall back to another account.
			if errors.Is(err, ErrNotAuthenticated) {
				t.Fatal("scope mismatch permits authentication fallback")
			}
			if _, err := store.GetSession(ctx, did, ""); !errors.Is(err, keyring.ErrNotFound) {
				t.Fatalf("stale session remains: %v", err)
			}
			if _, err := store.Account(did.String()); !errors.Is(err, keyring.ErrNotFound) {
				t.Fatalf("stale account remains: %v", err)
			}
			remaining, err := store.GetSession(ctx, other.AccountDID, "")
			if err != nil || !reflect.DeepEqual(*remaining, other) {
				t.Fatalf("other account changed: %v", err)
			}
		})
	}
}

func TestOAuthScopeValidationAcceptsCompatibleGrants(t *testing.T) {
	for _, name := range []string{"exact", "reordered and extra", "equivalent encoding"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := testKeyringStore(newFakeKeyring())
			manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
			session := fullyPopulatedSession(t, mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff"))
			switch name {
			case "reordered and extra":
				slices.Reverse(session.Scopes)
				session.Scopes = append(session.Scopes, "repo:sh.tangled.repo", "atproto")
			case "equivalent encoding":
				for index, scope := range session.Scopes {
					scope = strings.Replace(scope, "?aud=*", "?aud=%2A", 1)
					scope = strings.Replace(scope, "repo:", "repo?collection=", 1)
					session.Scopes[index] = scope
				}
			}
			if err := store.SaveSession(ctx, session); err != nil {
				t.Fatal(err)
			}
			if _, _, err := manager.APIClient(ctx); err != nil {
				t.Fatal(err)
			}
			hasScope, isOAuth, err := manager.OAuthSessionHasScope(ctx, "rpc:org.tangled.git.receivePack?aud=*")
			if err != nil || !hasScope || !isOAuth {
				t.Fatalf("scope query = %v, %v, %v", hasScope, isOAuth, err)
			}
			if _, err := store.GetSession(ctx, session.AccountDID, ""); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOAuthScopeValidationRejectsMissingRequiredGrants(t *testing.T) {
	for index, missing := range DefaultScopes {
		t.Run(missing, func(t *testing.T) {
			store := testKeyringStore(newFakeKeyring())
			manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
			session := fullyPopulatedSession(t, mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff"))
			session.Scopes = append(session.Scopes[:index], session.Scopes[index+1:]...)
			if err := store.SaveSession(context.Background(), session); err != nil {
				t.Fatal(err)
			}
			_, err := manager.CurrentSession(context.Background())
			if !errors.Is(err, ErrOAuthScopeMismatch) {
				t.Fatalf("missing scope accepted: %v", err)
			}
			if !strings.Contains(err.Error(), "tg auth login "+session.AccountDID.String()) {
				t.Fatalf("missing login hint: %v", err)
			}
		})
	}
}

func TestOAuthScopeMismatchReportsCredentialRemovalFailure(t *testing.T) {
	ctx := context.Background()
	backend := newFakeKeyring()
	store := testKeyringStore(backend)
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	session := fullyPopulatedSession(t, mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff"))
	session.Scopes = nil
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("keyring locked")
	backend.deleteErr = failure
	_, _, err := manager.APIClient(ctx)
	if !errors.Is(err, ErrOAuthScopeMismatch) || !errors.Is(err, failure) || strings.Contains(err.Error(), "signed out") {
		t.Fatalf("error = %v", err)
	}
	if _, err := store.GetSession(ctx, session.AccountDID, ""); err != nil {
		t.Fatalf("session changed after failed removal: %v", err)
	}
}

func TestOAuthScopeValidationPreservesSessionOnNetworkFailure(t *testing.T) {
	ctx := context.Background()
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	manager.app.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("network offline") })}
	session := fullyPopulatedSession(t, mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff"))
	if err := store.SaveSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	status, _, err := manager.SessionStatus(ctx)
	if err != nil || status != SessionStatusUnknown {
		t.Fatalf("status = %s, %v", status, err)
	}
	if _, err := store.GetSession(ctx, session.AccountDID, ""); err != nil {
		t.Fatalf("session removed after network failure: %v", err)
	}
}

func TestOAuthScopeValidationLeavesAppPasswordSessions(t *testing.T) {
	ctx := context.Background()
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	session := samplePasswordSession(mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff"), "https://pds.example")
	if err := store.SavePasswordSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	hasScope, isOAuth, err := manager.OAuthSessionHasScope(ctx, DefaultScopes[0])
	if err != nil || hasScope || isOAuth {
		t.Fatalf("scope query = %v, %v, %v", hasScope, isOAuth, err)
	}
	if _, _, err := manager.APIClient(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPasswordSession(ctx, session.AccountDID); err != nil {
		t.Fatal(err)
	}
}
