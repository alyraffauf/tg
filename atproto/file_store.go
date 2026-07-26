package atproto

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/zalando/go-keyring"
)

const (
	credentialsDirectoryMode = 0o700
	credentialsFileMode      = 0o600
	credentialsFileName      = "credentials.json"
)

// insecureFileStore persists a single app-password session to a plaintext file for
// environments without a Secret Service provider (e.g. headless servers). It is
// opt-in via `tg auth login --insecure` and stores exactly one account;
// re-login overwrites the previous entry.
type insecureFileStore struct {
	path string
	mu   sync.Mutex
}

// credentialRecord is the on-disk JSON shape: the account handle alongside the
// session, so the active account can be reported without resolving the DID.
type credentialRecord struct {
	Handle  string                       `json:"handle"`
	Session atclient.PasswordSessionData `json:"session"`
}

// newInsecureFileStore creates a store rooted at $XDG_DATA_HOME/tg/credentials.json
// (default ~/.local/share/tg/credentials.json), creating the directory with
// 0700 permissions. Returns an error if the home directory cannot be located or
// the directory cannot be created.
func newInsecureFileStore() (*insecureFileStore, error) {
	dir, err := insecureCredentialsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, credentialsDirectoryMode); err != nil {
		return nil, fmt.Errorf("create credentials directory: %w", err)
	}
	return &insecureFileStore{path: filepath.Join(dir, credentialsFileName)}, nil
}

func insecureCredentialsDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "tg"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "tg"), nil
}

// load reads and decodes the credential file. A missing file is reported as
// keyring.ErrNotFound so callers can share the same not-found handling as the
// keyring path.
func (s *insecureFileStore) load() (*credentialRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, keyring.ErrNotFound
		}
		return nil, fmt.Errorf("read credentials file: %w", err)
	}
	var record credentialRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse credentials file: %w", err)
	}
	return &record, nil
}

// save writes the record to disk with restrictive permissions, replacing any existing entry.
func (s *insecureFileStore) save(record credentialRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	if err := os.WriteFile(s.path, data, credentialsFileMode); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	return nil
}

// GetPasswordSession returns the stored session and the handle it was saved
// under.
func (s *insecureFileStore) GetPasswordSession() (*atclient.PasswordSessionData, string, error) {
	record, err := s.load()
	if err != nil {
		return nil, "", err
	}
	session := record.Session
	return &session, record.Handle, nil
}

// FindAccount returns the account described by the stored app-password session.
func (s *insecureFileStore) FindAccount() (Account, bool, error) {
	session, handle, err := s.GetPasswordSession()
	if errors.Is(err, keyring.ErrNotFound) {
		return Account{}, false, nil
	}
	if err != nil {
		return Account{}, false, err
	}
	return Account{DID: session.AccountDID.String(), Handle: handle, Method: AuthMethodPassword}, true, nil
}

// SavePasswordSession writes the session for the given handle, overwriting any
// existing entry.
func (s *insecureFileStore) SavePasswordSession(handle string, session atclient.PasswordSessionData) error {
	return s.save(credentialRecord{Handle: handle, Session: session})
}

// DeletePasswordSession removes the credential file. A missing file is not an
// error.
func (s *insecureFileStore) DeletePasswordSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete credentials file: %w", err)
	}
	return nil
}
