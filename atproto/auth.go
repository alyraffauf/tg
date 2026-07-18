package atproto

import (
	"context"
	"errors"
	"net/url"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

var ErrNotAuthenticated = errors.New("not authenticated")

// DefaultScopes are requested for a CLI session. The rpc scopes are needed for the
// PDS to mint service-auth JWTs for knot procedures. The blob scope is
// required for uploading PR patch blobs to the PDS.
var DefaultScopes = []string{
	"atproto",
	"repo:sh.tangled.actor.profile",
	"repo:sh.tangled.repo.issue.comment",
	"repo:sh.tangled.repo.issue.state",
	"repo:sh.tangled.repo.pull.comment",
	"repo:sh.tangled.repo.pull.status",
	"repo:sh.tangled.feed.star",
	"repo:sh.tangled.graph.follow",
	"repo:sh.tangled.graph.vouch",
	"repo:sh.tangled.publicKey",
	"repo:sh.tangled.repo",
	"repo:sh.tangled.repo.issue",
	"repo:sh.tangled.repo.pull",
	"blob:application/gzip",
	"rpc:sh.tangled.repo.create?aud=*",
	"rpc:sh.tangled.repo.delete?aud=*",
	"rpc:sh.tangled.repo.merge?aud=*",
	"rpc:sh.tangled.repo.setDefaultBranch?aud=*",
}

type AuthManager struct {
	app   *oauth.ClientApp
	store *KeyringStore
}

func NewAuthManager(callbackURL string) *AuthManager {
	config := oauth.NewLocalhostConfig(callbackURL, DefaultScopes)
	config.UserAgent = "tg"
	store := NewKeyringStore()
	return &AuthManager{
		app:   oauth.NewClientApp(&config, store),
		store: store,
	}
}

func (m *AuthManager) StartLogin(ctx context.Context, identifier string) (string, error) {
	return m.app.StartAuthFlow(ctx, identifier)
}

func (m *AuthManager) FinishLogin(ctx context.Context, query url.Values) error {
	_, err := m.app.ProcessCallback(ctx, query)
	return err
}

// CancelLogin cleans up any pending auth request written by StartLogin when the
// login flow is abandoned (e.g. the user closes the browser before the
// callback). It is safe to call after a completed login.
func (m *AuthManager) CancelLogin() {
	_ = m.store.DeletePendingAuthRequest()
}

func (m *AuthManager) CurrentDID(ctx context.Context) (syntax.DID, error) {
	session, err := m.CurrentSession(ctx)
	if err != nil {
		return "", err
	}
	return session.Data.AccountDID, nil
}

func (m *AuthManager) CurrentSession(ctx context.Context) (*oauth.ClientSession, error) {
	session, err := m.app.ResumeSession(ctx, "", "")
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrNotAuthenticated
		}
		return nil, err
	}
	return session, nil
}

func (m *AuthManager) Logout(ctx context.Context) error {
	err := m.app.Logout(ctx, "", "")
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotAuthenticated
	}
	// Logout failed partway — most commonly because the session entry is
	// corrupt or the keyring is transiently unavailable. The upstream Logout
	// aborts before DeleteSession in that case, so the bad entry would
	// otherwise be unrecoverable short of manual keyring surgery. Best-effort
	// clear it so the user can re-login; if that also fails, surface the
	// original error.
	if deleteErr := m.store.DeleteSession(ctx, "", ""); deleteErr == nil {
		return nil
	}
	return err
}
