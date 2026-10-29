package atproto

import (
	"context"
	"testing"
)

func TestDefaultScopesIncludeFeedComment(t *testing.T) {
	for _, scope := range DefaultScopes {
		if scope == "rpc:com.atproto.repo.createRecord?aud=*" {
			return
		}
	}
	t.Fatal("DefaultScopes does not include rpc:com.atproto.repo.createRecord?aud=*")
}

func TestDefaultScopesExcludeDeprecatedCommentCollections(t *testing.T) {
	deprecatedScopes := map[string]bool{
		"repo:sh.tangled.repo.issue.comment": true,
		"repo:sh.tangled.repo.pull.comment":  true,
	}
	for _, scope := range DefaultScopes {
		if deprecatedScopes[scope] {
			t.Fatalf("DefaultScopes includes deprecated scope %q", scope)
		}
	}
}

func TestAuthManagerOAuthSessionHasScope(t *testing.T) {
	store := testKeyringStore(newFakeKeyring())
	manager := newAuthManagerForTest("http://127.0.0.1:8095/callback", store)
	did := mustDID(t, "did:plc:aaaabbbbccccddddeeeeffff")
	session := fullyPopulatedSession(t, did)
	session.Scopes = append(append([]string(nil), DefaultScopes...), "rpc:sh.tangled.repo.push?aud=*")
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

func TestDefaultScopesCoverCurrentCIMutations(t *testing.T) {
	for _, method := range []string{"org.tangled.event.notifyEvent", "org.tangled.ci.approvePipeline", "org.tangled.ci.retryPipeline", "org.tangled.ci.cancelPipeline"} {
		found := false
		for _, scope := range DefaultScopes {
			found = found || scope == "rpc:"+method+"?aud=*"
		}
		if !found {
			t.Errorf("missing scope for %s", method)
		}
	}
	for _, scope := range DefaultScopes {
		if scope == "rpc:org.tangled.ci.triggerPipeline?aud=*" {
			t.Fatal("requested obsolete trigger scope")
		}
	}
}
