package atproto

import (
	"context"
	"testing"
)

func TestDefaultScopesIncludeFeedComment(t *testing.T) {
	for _, scope := range DefaultScopes {
		if scope == "repo:sh.tangled.feed.comment" {
			return
		}
	}
	t.Fatal("DefaultScopes does not include repo:sh.tangled.feed.comment")
}

func TestAuthManagerOAuthSessionHasScope(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	session := fullyPopulatedSession(t, did)
	session.Scopes = []string{"atproto", "rpc:sh.tangled.repo.push?aud=*"}
	if err := store.SaveSession(context.Background(), session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	hasScope, isOAuth, err := manager.OAuthSessionHasScope(context.Background(), "rpc:sh.tangled.repo.push?aud=*")
	if err != nil {
		t.Fatalf("OAuthSessionHasScope() error = %v", err)
	}
	if !isOAuth || !hasScope {
		t.Fatalf("OAuthSessionHasScope() = (%t, %t), want (true, true)", hasScope, isOAuth)
	}
}
