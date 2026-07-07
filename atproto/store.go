package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// FileStore implements oauth.ClientAuthStore on the local filesystem.
type FileStore struct {
	Directory string
}

func NewFileStore(dir string) *FileStore {
	return &FileStore{Directory: dir}
}

func (s *FileStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
	data, err := os.ReadFile(s.sessionPath(did, sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %w", err)
		}
		return nil, err
	}

	var session oauth.ClientSessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return &session, nil
}

func (s *FileStore) SaveSession(ctx context.Context, session oauth.ClientSessionData) error {
	path := s.sessionPath(session.AccountDID, session.SessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *FileStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	return os.Remove(s.sessionPath(did, sessionID))
}

func (s *FileStore) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	data, err := os.ReadFile(s.requestPath(state))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("auth request not found: %w", err)
		}
		return nil, err
	}

	var info oauth.AuthRequestData
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("decode auth request: %w", err)
	}
	return &info, nil
}

func (s *FileStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	path := s.requestPath(info.State)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *FileStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	return os.Remove(s.requestPath(state))
}

func (s *FileStore) sessionPath(did syntax.DID, sessionID string) string {
	return filepath.Join(s.Directory, "sessions", did.String(), sessionID+".json")
}

func (s *FileStore) requestPath(state string) string {
	return filepath.Join(s.Directory, "requests", state+".json")
}
