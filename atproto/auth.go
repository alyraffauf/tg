package atproto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrNotAuthenticated = errors.New("not authenticated")

// DefaultScopes are requested for a CLI session. The rpc scopes are
// needed for the PDS to mint service-auth JWTs for knot procedures. The
// blob scope is required for uploading PR patch blobs to the PDS.
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
	App             *oauth.ClientApp
	Store           *FileStore
	state           authState
	statePath       string
	passwordPath    string
	passwordSession *atclient.PasswordSessionData
}

type authState struct {
	CurrentDID     string `json:"current_did,omitempty"`
	CurrentSession string `json:"current_session,omitempty"`
	Method         string `json:"method,omitempty"`
}

// ConfigDir returns the configuration directory for tg.
//
// It respects the XDG Base Directory Specification: if XDG_CONFIG_HOME is set,
// it uses that directory; otherwise it falls back to ~/.config/tg.
func ConfigDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "tg"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "tg"), nil
}

// NewAuthManager creates an AuthManager. callbackURL must be reachable by the
// user's browser during login.
func NewAuthManager(callbackURL string, dir string) (*AuthManager, error) {
	config := oauth.NewLocalhostConfig(callbackURL, DefaultScopes)
	config.UserAgent = "tg"

	store := NewFileStore(filepath.Join(dir, "oauth"))
	manager := &AuthManager{
		App:          oauth.NewClientApp(&config, store),
		Store:        store,
		statePath:    filepath.Join(dir, "auth.json"),
		passwordPath: filepath.Join(dir, "password-session.json"),
	}
	if err := manager.loadState(); err != nil {
		return nil, fmt.Errorf("load auth state: %w", err)
	}
	if manager.state.Method == "password" {
		data, err := os.ReadFile(manager.passwordPath)
		if err != nil {
			manager.state = authState{}
			if err := manager.saveState(); err != nil {
				return nil, fmt.Errorf("clear unusable password auth state: %w", err)
			}
			return manager, nil
		}
		var session atclient.PasswordSessionData
		if err := json.Unmarshal(data, &session); err != nil {
			if removeErr := os.Remove(manager.passwordPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return nil, fmt.Errorf("remove unusable password session: %w", removeErr)
			}
			manager.state = authState{}
			if err := manager.saveState(); err != nil {
				return nil, fmt.Errorf("clear unusable password auth state: %w", err)
			}
			return manager, nil
		}
		if session.AccountDID == "" || session.Host == "" || session.RefreshToken == "" {
			if err := os.Remove(manager.passwordPath); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("remove incomplete password session: %w", err)
			}
			manager.state = authState{}
			if err := manager.saveState(); err != nil {
				return nil, fmt.Errorf("clear incomplete password auth state: %w", err)
			}
			return manager, nil
		}
		manager.passwordSession = &session
	}
	return manager, nil
}

// LoginWithPassword authenticates with an atproto app password and persists
// the resulting access/refresh token pair for subsequent invocations.
func (m *AuthManager) LoginWithPassword(ctx context.Context, identifier, password string) error {
	atid, err := syntax.ParseAtIdentifier(identifier)
	if err != nil {
		return err
	}
	persistSession := func(_ context.Context, data atclient.PasswordSessionData) {
		_ = m.savePasswordSession(&data)
	}
	client, err := atclient.LoginWithPassword(
		ctx,
		identity.DefaultDirectory(),
		atid,
		password,
		"",
		persistSession,
	)
	if err != nil {
		return err
	}
	if client.Auth == nil {
		return errors.New("password login returned no auth session")
	}
	passwordAuth, ok := client.Auth.(*atclient.PasswordAuth)
	if !ok {
		return errors.New("password login returned an unexpected auth type")
	}
	if m.IsAuthenticated() {
		if err := m.Logout(ctx); err != nil {
			return fmt.Errorf("replace current login: %w", err)
		}
	}
	if err := m.savePasswordSession(&passwordAuth.Session); err != nil {
		return err
	}
	m.state = authState{
		CurrentDID: passwordAuth.Session.AccountDID.String(),
		Method:     "password",
	}
	return m.saveState()
}

func (m *AuthManager) savePasswordSession(session *atclient.PasswordSessionData) error {
	snapshot := *session
	m.passwordSession = &snapshot
	if err := os.MkdirAll(filepath.Dir(m.passwordPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(&snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.passwordPath, data, 0o600)
}

func (m *AuthManager) StartLogin(ctx context.Context, identifier string) (string, error) {
	return m.App.StartAuthFlow(ctx, identifier)
}

func (m *AuthManager) FinishLogin(ctx context.Context, query url.Values) error {
	session, err := m.App.ProcessCallback(ctx, query)
	if err != nil {
		return err
	}
	if m.IsAuthenticated() {
		if err := m.Logout(ctx); err != nil {
			return fmt.Errorf("replace current login: %w", err)
		}
	}

	return m.activateOAuthSession(session.AccountDID.String(), session.SessionID)
}

func (m *AuthManager) activateOAuthSession(did, sessionID string) error {
	if err := os.Remove(m.passwordPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove previous password session: %w", err)
	}
	m.passwordSession = nil
	m.state = authState{
		CurrentDID:     did,
		CurrentSession: sessionID,
		Method:         "oauth",
	}
	return m.saveState()
}

func (m *AuthManager) CurrentDID() syntax.DID {
	if m.state.CurrentDID == "" {
		return syntax.DID("")
	}
	did, err := syntax.ParseDID(m.state.CurrentDID)
	if err != nil {
		return syntax.DID("")
	}
	return did
}

func (m *AuthManager) IsAuthenticated() bool {
	return m.state.CurrentDID != "" && (m.state.CurrentSession != "" || m.passwordSession != nil)
}

func (m *AuthManager) CurrentSession(ctx context.Context) (*oauth.ClientSession, error) {
	if !m.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return m.App.ResumeSession(ctx, m.CurrentDID(), m.state.CurrentSession)
}

func (m *AuthManager) APIClient(ctx context.Context) (*atclient.APIClient, error) {
	if m.state.Method == "password" && m.passwordSession != nil {
		persistSession := func(_ context.Context, data atclient.PasswordSessionData) {
			_ = m.savePasswordSession(&data)
		}
		return atclient.ResumePasswordSession(*m.passwordSession, persistSession), nil
	}
	session, err := m.CurrentSession(ctx)
	if err != nil {
		return nil, err
	}
	return session.APIClient(), nil
}

func (m *AuthManager) Logout(ctx context.Context) error {
	if !m.IsAuthenticated() {
		return nil
	}
	if m.state.Method == "password" {
		client := atclient.ResumePasswordSession(*m.passwordSession, nil)
		passwordAuth, ok := client.Auth.(*atclient.PasswordAuth)
		if !ok {
			return errors.New("password session has an unexpected auth type")
		}
		if err := passwordAuth.Logout(ctx, client.Client); err != nil {
			return fmt.Errorf("revoke password session: %w", err)
		}
		if err := os.Remove(m.passwordPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove password session: %w", err)
		}
		m.passwordSession = nil
		m.state = authState{}
		return m.saveState()
	}
	if err := m.App.Logout(ctx, m.CurrentDID(), m.state.CurrentSession); err != nil {
		return err
	}

	m.state = authState{}
	return m.saveState()
}

func (m *AuthManager) loadState() error {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &m.state)
}

func (m *AuthManager) saveState() error {
	if err := os.MkdirAll(filepath.Dir(m.statePath), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.statePath, data, 0o600)
}
