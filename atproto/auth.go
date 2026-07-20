package atproto

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

var ErrNotAuthenticated = errors.New("not authenticated")

const (
	SessionStatusActive  = "active"
	SessionStatusExpired = "expired"
	SessionStatusUnknown = "unknown"
)

// DefaultScopes are requested for a CLI session. The rpc scopes are needed for
// the PDS to mint service-auth JWTs for knot procedures. The blob scope is
// required for uploading PR patch blobs to the PDS.
var DefaultScopes = []string{
	"atproto",

	// Record collections.
	"repo:sh.tangled.actor.profile",
	"repo:sh.tangled.feed.comment",
	"repo:sh.tangled.feed.reaction",
	"repo:sh.tangled.feed.star",
	"repo:sh.tangled.graph.follow",
	"repo:sh.tangled.graph.vouch",
	"repo:sh.tangled.knot",
	"repo:sh.tangled.knot.member",
	"repo:sh.tangled.label.definition",
	"repo:sh.tangled.label.op",
	"repo:sh.tangled.publicKey",
	"repo:sh.tangled.repo",
	"repo:sh.tangled.repo.artifact",
	"repo:sh.tangled.repo.collaborator",
	"repo:sh.tangled.repo.issue",
	"repo:sh.tangled.repo.issue.comment",
	"repo:sh.tangled.repo.issue.state",
	"repo:sh.tangled.repo.pull",
	"repo:sh.tangled.repo.pull.comment",
	"repo:sh.tangled.repo.pull.status",
	"repo:sh.tangled.spindle",
	"repo:sh.tangled.spindle.member",
	"repo:sh.tangled.string",

	// Blob uploads: gzipped git patches plus image/video media.
	"blob:application/gzip",
	"blob:image/*",
	"blob:video/*",

	// RPC procedures, addressed to any service ("aud=*").
	"rpc:sh.tangled.ci.cancelPipeline?aud=*",
	"rpc:sh.tangled.ci.triggerPipeline?aud=*",
	"rpc:sh.tangled.knot.addMember?aud=*",
	"rpc:sh.tangled.knot.removeMember?aud=*",
	"rpc:sh.tangled.repo.addCollaborator?aud=*",
	"rpc:sh.tangled.repo.addSecret?aud=*",
	"rpc:sh.tangled.repo.create?aud=*",
	"rpc:sh.tangled.repo.delete?aud=*",
	"rpc:sh.tangled.repo.deleteBranch?aud=*",
	"rpc:sh.tangled.repo.forkStatus?aud=*",
	"rpc:sh.tangled.repo.forkSync?aud=*",
	"rpc:sh.tangled.repo.hiddenRef?aud=*",
	"rpc:sh.tangled.repo.listSecrets?aud=*",
	"rpc:sh.tangled.repo.merge?aud=*",
	"rpc:sh.tangled.repo.mergeCheck?aud=*",
	"rpc:sh.tangled.repo.removeCollaborator?aud=*",
	"rpc:sh.tangled.repo.removeSecret?aud=*",
	"rpc:sh.tangled.repo.setDefaultBranch?aud=*",
}

type AuthManager struct {
	app               *oauth.ClientApp
	store             *KeyringStore
	selector          string
	pendingIdentifier string
}

func (m *AuthManager) SetAccount(selector string) {
	m.selector = selector
}

func (m *AuthManager) Accounts() ([]Account, string, error) {
	return m.store.Accounts()
}

func (m *AuthManager) SelectAccount(selector string) (Account, error) {
	return m.store.SelectAccount(selector)
}

func (m *AuthManager) activeAccount() (Account, error) {
	account, err := m.store.Account(m.selector)
	if errors.Is(err, keyring.ErrNotFound) {
		return Account{}, ErrNotAuthenticated
	}
	return account, err
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
// only one auth method is active for this account.
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
	if err := m.store.SavePasswordSession(ctx, passwordAuth.Session); err != nil {
		return err
	}
	did := passwordAuth.Session.AccountDID.String()
	if err := m.store.SetAccountHandle(did, identifier); err != nil {
		return err
	}
	_, err = m.store.SelectAccount(did)
	return err
}

func (m *AuthManager) StartLogin(ctx context.Context, identifier string) (string, error) {
	loginURL, err := m.app.StartAuthFlow(ctx, identifier)
	if err == nil {
		m.pendingIdentifier = identifier
	}
	return loginURL, err
}

func (m *AuthManager) FinishLogin(ctx context.Context, query url.Values) error {
	session, err := m.app.ProcessCallback(ctx, query)
	if err != nil {
		return err
	}
	handle := m.pendingIdentifier
	if handle == "" {
		handle = session.AccountDID.String()
	}
	m.pendingIdentifier = ""
	did := session.AccountDID.String()
	if err := m.store.SetAccountHandle(did, handle); err != nil {
		return err
	}
	_, err = m.store.SelectAccount(did)
	return err
}

// CancelLogin cleans up any pending auth request written by StartLogin when the
// login flow is abandoned (e.g. the user closes the browser before the
// callback). It is safe to call after a completed login.
func (m *AuthManager) CancelLogin() {
	m.pendingIdentifier = ""
	_ = m.store.DeletePendingAuthRequest()
}

func (m *AuthManager) CurrentDID(ctx context.Context) (syntax.DID, error) {
	account, err := m.activeAccount()
	if err != nil {
		return "", err
	}
	did, err := syntax.ParseDID(account.DID)
	if err != nil {
		return "", err
	}
	if account.Method == AuthMethodOAuth {
		session, err := m.app.ResumeSession(ctx, did, "")
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				return "", ErrNotAuthenticated
			}
			return "", err
		}
		return session.Data.AccountDID, nil
	}
	passwordSession, err := m.store.GetPasswordSession(ctx, did)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotAuthenticated
		}
		return "", err
	}
	return passwordSession.AccountDID, nil
}

func (m *AuthManager) CurrentSession(ctx context.Context) (*oauth.ClientSession, error) {
	account, err := m.activeAccount()
	if err != nil {
		return nil, err
	}
	if account.Method != AuthMethodOAuth {
		return nil, ErrNotAuthenticated
	}
	did, err := syntax.ParseDID(account.DID)
	if err != nil {
		return nil, err
	}
	session, err := m.app.ResumeSession(ctx, did, "")
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
	account, err := m.activeAccount()
	if err != nil {
		return nil, "", err
	}
	did, err := syntax.ParseDID(account.DID)
	if err != nil {
		return nil, "", err
	}
	if account.Method == AuthMethodOAuth {
		session, err := m.app.ResumeSession(ctx, did, "")
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				return nil, "", ErrNotAuthenticated
			}
			return nil, "", err
		}
		return session.APIClient(), session.Data.AccountDID, nil
	}
	passwordSession, err := m.store.GetPasswordSession(ctx, did)
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

// SessionStatus probes the server to verify the active session. The probe
// refreshes the access token if needed. Returns ErrNotAuthenticated when
// there is no active account.
func (m *AuthManager) SessionStatus(ctx context.Context) (string, syntax.DID, error) {
	client, did, err := m.APIClient(ctx)
	if err != nil {
		return "", "", err
	}
	err = client.Get(ctx, syntax.NSID("com.atproto.server.getSession"), nil, nil)
	if err == nil {
		return SessionStatusActive, did, nil
	}
	var apiError *atclient.APIError
	if errors.As(err, &apiError) && apiError.StatusCode == http.StatusUnauthorized {
		return SessionStatusExpired, did, nil
	}
	// OAuth wraps refresh failures in fmt.Errorf, not APIError. A failed
	// refresh means the refresh token is dead.
	if strings.Contains(err.Error(), "failed to refresh OAuth tokens") {
		return SessionStatusExpired, did, nil
	}
	return SessionStatusUnknown, did, nil
}

// Logout removes the active account's credentials from the local keyring.
// Server-side revocation is best-effort; the local entry is removed even
// if the PDS rejects the revoke request. Returns ErrNotAuthenticated when
// there is no active account.
func (m *AuthManager) Logout(ctx context.Context) error {
	account, err := m.activeAccount()
	if err != nil {
		return err
	}
	did, err := syntax.ParseDID(account.DID)
	if err != nil {
		return err
	}
	switch account.Method {
	case AuthMethodOAuth:
		_ = m.app.Logout(ctx, did, "")
		if err := m.store.DeleteSession(ctx, did, ""); err != nil {
			return fmt.Errorf("clear local session: %w", err)
		}
	case AuthMethodPassword:
		passwordSession, err := m.store.GetPasswordSession(ctx, did)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return err
		}
		if passwordSession != nil {
			client := atclient.ResumePasswordSession(*passwordSession, nil)
			if passwordAuth, ok := client.Auth.(*atclient.PasswordAuth); ok {
				_ = passwordAuth.Logout(ctx, client.Client)
			}
		}
		if err := m.store.DeletePasswordSession(ctx, did); err != nil {
			return fmt.Errorf("clear local session: %w", err)
		}
	default:
		return fmt.Errorf("unsupported auth method %q", account.Method)
	}
	return nil
}

func (m *AuthManager) LogoutAll(ctx context.Context) error {
	accounts, _, err := m.store.Accounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		return ErrNotAuthenticated
	}
	originalSelector := m.selector
	defer func() { m.selector = originalSelector }()
	var errs []error
	for _, account := range accounts {
		m.selector = account.DID
		if err := m.Logout(ctx); err != nil {
			errs = append(errs, fmt.Errorf("logout %s: %w", account.DID, err))
		}
	}
	return errors.Join(errs...)
}
