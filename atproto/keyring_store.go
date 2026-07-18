package atproto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

// Reverse-DNS of the repo so it won't collide with other clients.
const keyringService = "io.github.alyraffauf.tg"

// Interface over go-keyring so tests can inject a fake.
type secretBackend interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type goKeyringBackend struct{}

func (goKeyringBackend) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (goKeyringBackend) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (goKeyringBackend) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

// KeyringStore implements oauth.ClientAuthStore on top of the OS keyring.
//
// A mutex serializes access within one process. It does NOT coordinate across
// processes: concurrent invocations that refresh tokens can race, and the loser
// may need to re-login.
type KeyringStore struct {
	backend secretBackend
	service string

	mu sync.Mutex
	// Most recent auth-request state, tracked so an abandoned login can clean up.
	pendingState string
}

func NewKeyringStore() *KeyringStore {
	return &KeyringStore{backend: goKeyringBackend{}, service: keyringService}
}

const currentSessionKey = "session:current"

func requestKey(state string) string {
	return "request:" + state
}

func (s *KeyringStore) getSecret(key string, target any) error {
	data, err := s.backend.Get(s.service, key)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(data), target); err != nil {
		return fmt.Errorf("decode secret %q: %w", key, err)
	}
	return nil
}

func (s *KeyringStore) saveSecret(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	return s.backend.Set(s.service, key, string(data))
}

// Ignore not-found errors; the entry is already gone.
func (s *KeyringStore) deleteSecret(key string) error {
	err := s.backend.Delete(s.service, key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

func (s *KeyringStore) GetSession(_ context.Context, _ syntax.DID, _ string) (*oauth.ClientSessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var session oauth.ClientSessionData
	if err := s.getSecret(currentSessionKey, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *KeyringStore) SaveSession(_ context.Context, session oauth.ClientSessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveSecret(currentSessionKey, session)
}

func (s *KeyringStore) DeleteSession(_ context.Context, _ syntax.DID, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteSecret(currentSessionKey)
}

func (s *KeyringStore) GetAuthRequestInfo(_ context.Context, state string) (*oauth.AuthRequestData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var info oauth.AuthRequestData
	if err := s.getSecret(requestKey(state), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *KeyringStore) SaveAuthRequestInfo(_ context.Context, info oauth.AuthRequestData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingState = info.State
	return s.saveSecret(requestKey(info.State), info)
}

func (s *KeyringStore) DeleteAuthRequestInfo(_ context.Context, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingState == state {
		s.pendingState = ""
	}
	return s.deleteSecret(requestKey(state))
}

// DeletePendingAuthRequest removes the auth-request entry written by the most
// recent SaveAuthRequestInfo, if it hasn't already been consumed. Used to clean
// up when a login is abandoned before the callback completes. Missing entries
// are ignored.
func (s *KeyringStore) DeletePendingAuthRequest() error {
	s.mu.Lock()
	state := s.pendingState
	s.pendingState = ""
	s.mu.Unlock()
	if state == "" {
		return nil
	}
	return s.deleteSecret(requestKey(state))
}
