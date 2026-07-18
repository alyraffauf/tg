package atproto

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

var ErrNotAuthenticated = errors.New("not authenticated")

// DefaultScopes are requested for a CLI session. The rpc scopes are needed for
// the PDS to mint service-auth JWTs for knot procedures. The blob scope is
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

// LoginWithPassword authenticates with an atproto app password and stores the
// resulting session in the keyring. Any existing OAuth session is cleared so
// only one auth method is active at a time.
func (m *AuthManager) LoginWithPassword(ctx context.Context, identifier, password string) error {
	parsedIdentifier, err := syntax.ParseAtIdentifier(identifier)
	if err != nil {
		return err
	}
	persist := func(_ context.Context, data atclient.PasswordSessionData) {
		_ = m.store.SavePasswordSession(context.Background(), data)
	}
	client, err := atclient.LoginWithPassword(ctx, identity.DefaultDirectory(), parsedIdentifier, password, "", persist)
	if err != nil {
		return err
	}
	passwordAuth, ok := client.Auth.(*atclient.PasswordAuth)
	if !ok {
		return errors.New("password login returned an unexpected auth type")
	}
	_ = m.store.DeleteSession(ctx, "", "")
	return m.store.SavePasswordSession(ctx, passwordAuth.Session)
}

func (m *AuthManager) StartLogin(ctx context.Context, identifier string) (string, error) {
	return m.app.StartAuthFlow(ctx, identifier)
}

func (m *AuthManager) FinishLogin(ctx context.Context, query url.Values) error {
	_, err := m.app.ProcessCallback(ctx, query)
	if err != nil {
		return err
	}
	// Clear any existing password session so only one auth method is active.
	_ = m.store.DeletePasswordSession(ctx)
	return nil
}

// CancelLogin cleans up any pending auth request written by StartLogin when the
// login flow is abandoned (e.g. the user closes the browser before the
// callback). It is safe to call after a completed login.
func (m *AuthManager) CancelLogin() {
	_ = m.store.DeletePendingAuthRequest()
}

func (m *AuthManager) CurrentDID(ctx context.Context) (syntax.DID, error) {
	session, err := m.app.ResumeSession(ctx, "", "")
	if err == nil {
		return session.Data.AccountDID, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return "", err
	}
	passwordSession, err := m.store.GetPasswordSession(ctx)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotAuthenticated
		}
		return "", err
	}
	return passwordSession.AccountDID, nil
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

// APIClient returns an API client and the account DID for the active session,
// whether OAuth or app-password. Token refreshes are persisted back to the
// keyring.
func (m *AuthManager) APIClient(ctx context.Context) (*atclient.APIClient, syntax.DID, error) {
	session, err := m.app.ResumeSession(ctx, "", "")
	if err == nil {
		return session.APIClient(), session.Data.AccountDID, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return nil, "", err
	}
	passwordSession, err := m.store.GetPasswordSession(ctx)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, "", ErrNotAuthenticated
		}
		return nil, "", err
	}
	persist := func(_ context.Context, data atclient.PasswordSessionData) {
		_ = m.store.SavePasswordSession(context.Background(), data)
	}
	client := atclient.ResumePasswordSession(*passwordSession, persist)
	return client, passwordSession.AccountDID, nil
}

func (m *AuthManager) Logout(ctx context.Context) error {
	err := m.app.Logout(ctx, "", "")
	switch {
	case err == nil:
		return nil
	case errors.Is(err, keyring.ErrNotFound):
		// No OAuth session; continue to password logout below.
	default:
		// Corrupt or transient OAuth failure — force clear so the user can
		// re-login instead of being locked out.
		if deleteErr := m.store.DeleteSession(ctx, "", ""); deleteErr == nil {
			return nil
		}
		return err
	}

	passwordSession, err := m.store.GetPasswordSession(ctx)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNotAuthenticated
		}
		return err
	}
	client := atclient.ResumePasswordSession(*passwordSession, nil)
	passwordAuth, ok := client.Auth.(*atclient.PasswordAuth)
	if !ok {
		// Corrupt password session — force clear.
		_ = m.store.DeletePasswordSession(ctx)
		return nil
	}
	if err := passwordAuth.Logout(ctx, client.Client); err != nil {
		return fmt.Errorf("revoke password session: %w", err)
	}
	return m.store.DeletePasswordSession(ctx)
}
