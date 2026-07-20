package atproto

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

const compressedSecretPrefix = "gzip:"

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

const currentPasswordKey = "password:current"

const accountIndexKey = "accounts:index"

const (
	AuthMethodOAuth    = "oauth"
	AuthMethodPassword = "password"
)

type Account struct {
	DID    string `json:"did"`
	Handle string `json:"handle,omitempty"`
	Method string `json:"method"`
}

type accountIndex struct {
	ActiveDID string    `json:"activeDid,omitempty"`
	Accounts  []Account `json:"accounts"`
}

func sessionKey(did string) string { return "oauth:" + did }

func passwordKey(did string) string { return "password:" + did }

func requestKey(state string) string {
	return "request:" + state
}

func (s *KeyringStore) getSecret(key string, target any) error {
	data, err := s.backend.Get(s.service, key)
	if err != nil {
		return err
	}
	decoded := []byte(data)
	if strings.HasPrefix(data, compressedSecretPrefix) {
		compressed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, compressedSecretPrefix))
		if err != nil {
			return fmt.Errorf("decode compressed secret %q: %w", key, err)
		}
		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return fmt.Errorf("open compressed secret %q: %w", key, err)
		}
		decoded, err = io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			return fmt.Errorf("read compressed secret %q: %w", key, err)
		}
		if closeErr != nil {
			return fmt.Errorf("close compressed secret %q: %w", key, closeErr)
		}
	}
	if err := json.Unmarshal(decoded, target); err != nil {
		return fmt.Errorf("decode secret %q: %w", key, err)
	}
	return nil
}

func (s *KeyringStore) saveSecret(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	if len(data) > 1024 {
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("compress value: %w", err)
		}
		if err := writer.Close(); err != nil {
			return fmt.Errorf("finish compressed value: %w", err)
		}
		data = []byte(compressedSecretPrefix + base64.StdEncoding.EncodeToString(compressed.Bytes()))
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

func (s *KeyringStore) loadIndexLocked() (accountIndex, error) {
	var index accountIndex
	err := s.getSecret(accountIndexKey, &index)
	if err == nil {
		return index, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return accountIndex{}, err
	}
	return s.migrateLegacyLocked()
}

func (s *KeyringStore) migrateLegacyLocked() (accountIndex, error) {
	index := accountIndex{Accounts: []Account{}}
	var oauthSession oauth.ClientSessionData
	if err := s.getSecret(currentSessionKey, &oauthSession); err == nil {
		did := oauthSession.AccountDID.String()
		if err := s.saveSecret(sessionKey(did), oauthSession); err != nil {
			return accountIndex{}, err
		}
		index.Accounts = append(index.Accounts, Account{DID: did, Method: AuthMethodOAuth})
		index.ActiveDID = did
	} else if !errors.Is(err, keyring.ErrNotFound) {
		return accountIndex{}, err
	}

	var passwordSession atclient.PasswordSessionData
	if err := s.getSecret(currentPasswordKey, &passwordSession); err == nil {
		did := passwordSession.AccountDID.String()
		if err := s.saveSecret(passwordKey(did), passwordSession); err != nil {
			return accountIndex{}, err
		}
		if !slices.ContainsFunc(index.Accounts, func(a Account) bool { return a.DID == did }) {
			index.Accounts = append(index.Accounts, Account{DID: did, Method: AuthMethodPassword})
			if index.ActiveDID == "" {
				index.ActiveDID = did
			}
		}
	} else if !errors.Is(err, keyring.ErrNotFound) {
		return accountIndex{}, err
	}

	if len(index.Accounts) == 0 {
		return index, nil
	}
	if err := s.saveSecret(accountIndexKey, index); err != nil {
		return accountIndex{}, err
	}
	if err := s.deleteSecret(currentSessionKey); err != nil {
		return accountIndex{}, err
	}
	if err := s.deleteSecret(currentPasswordKey); err != nil {
		return accountIndex{}, err
	}
	return index, nil
}

func (s *KeyringStore) saveIndexLocked(index accountIndex) error {
	return s.saveSecret(accountIndexKey, index)
}

func (s *KeyringStore) upsertAccountLocked(index *accountIndex, account Account) {
	for i := range index.Accounts {
		if index.Accounts[i].DID == account.DID {
			if account.Handle == "" {
				account.Handle = index.Accounts[i].Handle
			}
			index.Accounts[i] = account
			return
		}
	}
	index.Accounts = append(index.Accounts, account)
	if index.ActiveDID == "" {
		index.ActiveDID = account.DID
	}
}

func (s *KeyringStore) removeAccountLocked(index *accountIndex, did, method string) {
	index.Accounts = slices.DeleteFunc(index.Accounts, func(a Account) bool {
		return a.DID == did && a.Method == method
	})
	if index.ActiveDID == did && len(index.Accounts) > 0 {
		index.ActiveDID = index.Accounts[0].DID
	} else if len(index.Accounts) == 0 {
		index.ActiveDID = ""
	}
}

func findAccount(index accountIndex, selector string) (Account, error) {
	if selector == "" {
		selector = index.ActiveDID
	}
	for _, account := range index.Accounts {
		if account.DID == selector || strings.EqualFold(account.Handle, selector) {
			return account, nil
		}
	}
	return Account{}, keyring.ErrNotFound
}

func (s *KeyringStore) Accounts() ([]Account, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return nil, "", err
	}
	return slices.Clone(index.Accounts), index.ActiveDID, nil
}

func (s *KeyringStore) Account(selector string) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return Account{}, err
	}
	return findAccount(index, selector)
}

func (s *KeyringStore) SelectAccount(selector string) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return Account{}, err
	}
	account, err := findAccount(index, selector)
	if err != nil {
		return Account{}, err
	}
	switch account.Method {
	case AuthMethodOAuth:
		var session oauth.ClientSessionData
		if err := s.getSecret(sessionKey(account.DID), &session); err != nil {
			return Account{}, err
		}
	case AuthMethodPassword:
		var session atclient.PasswordSessionData
		if err := s.getSecret(passwordKey(account.DID), &session); err != nil {
			return Account{}, err
		}
	default:
		return Account{}, fmt.Errorf("unsupported auth method %q", account.Method)
	}
	index.ActiveDID = account.DID
	return account, s.saveIndexLocked(index)
}

func (s *KeyringStore) SetAccountHandle(did, handle string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return err
	}
	for i := range index.Accounts {
		if index.Accounts[i].DID == did {
			index.Accounts[i].Handle = handle
			return s.saveIndexLocked(index)
		}
	}
	return keyring.ErrNotFound
}

func (s *KeyringStore) GetSession(_ context.Context, did syntax.DID, _ string) (*oauth.ClientSessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return nil, err
	}
	account, err := findAccount(index, did.String())
	if err != nil || account.Method != AuthMethodOAuth {
		return nil, keyring.ErrNotFound
	}
	var session oauth.ClientSessionData
	if err := s.getSecret(sessionKey(account.DID), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *KeyringStore) SaveSession(_ context.Context, session oauth.ClientSessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return err
	}
	did := session.AccountDID.String()
	if err := s.saveSecret(sessionKey(did), session); err != nil {
		return err
	}
	s.upsertAccountLocked(&index, Account{DID: did, Method: AuthMethodOAuth})
	if err := s.saveIndexLocked(index); err != nil {
		return err
	}
	return s.deleteSecret(passwordKey(did))
}

func (s *KeyringStore) DeleteSession(_ context.Context, did syntax.DID, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return err
	}
	account, err := findAccount(index, did.String())
	if err != nil || account.Method != AuthMethodOAuth {
		return nil
	}
	if err := s.deleteSecret(sessionKey(account.DID)); err != nil {
		return err
	}
	s.removeAccountLocked(&index, account.DID, AuthMethodOAuth)
	return s.saveIndexLocked(index)
}

func (s *KeyringStore) GetPasswordSession(_ context.Context, did syntax.DID) (*atclient.PasswordSessionData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return nil, err
	}
	account, err := findAccount(index, did.String())
	if err != nil || account.Method != AuthMethodPassword {
		return nil, keyring.ErrNotFound
	}
	var session atclient.PasswordSessionData
	if err := s.getSecret(passwordKey(account.DID), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *KeyringStore) SavePasswordSession(_ context.Context, session atclient.PasswordSessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return err
	}
	did := session.AccountDID.String()
	if err := s.saveSecret(passwordKey(did), session); err != nil {
		return err
	}
	s.upsertAccountLocked(&index, Account{DID: did, Method: AuthMethodPassword})
	if err := s.saveIndexLocked(index); err != nil {
		return err
	}
	return s.deleteSecret(sessionKey(did))
}

func (s *KeyringStore) DeletePasswordSession(_ context.Context, did syntax.DID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	index, err := s.loadIndexLocked()
	if err != nil {
		return err
	}
	account, err := findAccount(index, did.String())
	if err != nil || account.Method != AuthMethodPassword {
		return nil
	}
	if err := s.deleteSecret(passwordKey(account.DID)); err != nil {
		return err
	}
	s.removeAccountLocked(&index, account.DID, AuthMethodPassword)
	return s.saveIndexLocked(index)
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
