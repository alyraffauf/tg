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
	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrNotAuthenticated = errors.New("not authenticated")

// DefaultScopes are requested for a CLI session. Rather than the broad
// transition:generic scope (equivalent to an app password), we request
// granular permissions scoped to all Tangled collections preemptively,
// so future features don't require re-authentication. The rpc scopes are
// needed for the PDS to mint service-auth JWTs for knot procedures.
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
	"rpc:sh.tangled.repo.create?aud=*",
	"rpc:sh.tangled.repo.delete?aud=*",
	"rpc:sh.tangled.repo.merge?aud=*",
	"rpc:sh.tangled.repo.setDefaultBranch?aud=*",
}

type AuthManager struct {
	App       *oauth.ClientApp
	Store     *FileStore
	state     authState
	statePath string
}

type authState struct {
	CurrentDID     string `json:"current_did,omitempty"`
	CurrentSession string `json:"current_session,omitempty"`
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
		App:       oauth.NewClientApp(&config, store),
		Store:     store,
		statePath: filepath.Join(dir, "auth.json"),
	}
	if err := manager.loadState(); err != nil {
		return nil, fmt.Errorf("load auth state: %w", err)
	}
	return manager, nil
}

func (m *AuthManager) StartLogin(ctx context.Context, identifier string) (string, error) {
	return m.App.StartAuthFlow(ctx, identifier)
}

func (m *AuthManager) FinishLogin(ctx context.Context, query url.Values) error {
	session, err := m.App.ProcessCallback(ctx, query)
	if err != nil {
		return err
	}

	m.state.CurrentDID = session.AccountDID.String()
	m.state.CurrentSession = session.SessionID
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
	return m.state.CurrentDID != "" && m.state.CurrentSession != ""
}

func (m *AuthManager) CurrentSession(ctx context.Context) (*oauth.ClientSession, error) {
	if !m.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}
	return m.App.ResumeSession(ctx, m.CurrentDID(), m.state.CurrentSession)
}

func (m *AuthManager) APIClient(ctx context.Context) (*atclient.APIClient, error) {
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
