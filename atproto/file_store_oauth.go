package atproto

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/zalando/go-keyring"
)

var _ oauth.ClientAuthStore = (*insecureFileStore)(nil)

func (s *insecureFileStore) GetSession(_ context.Context, did syntax.DID, _ string) (*oauth.ClientSessionData, error) {
	record, err := s.load()
	if err != nil {
		return nil, err
	}
	if record.OAuth == nil || record.OAuth.AccountDID != did {
		return nil, keyring.ErrNotFound
	}
	return record.OAuth, nil
}

func (s *insecureFileStore) SaveSession(_ context.Context, session oauth.ClientSessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.loadLocked()
	if err != nil && !errors.Is(err, keyring.ErrNotFound) && !errors.Is(err, errCredentialsCorrupt) {
		return err
	}
	if record == nil || record.account().DID != session.AccountDID.String() || record.OAuth == nil {
		record = &credentialRecord{}
	}
	record.Session = nil
	record.OAuth = &session
	return s.saveLocked(*record)
}

func (s *insecureFileStore) DeleteSession(_ context.Context, did syntax.DID, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.loadLocked()
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if record.OAuth == nil || record.OAuth.AccountDID != did {
		return nil
	}
	return s.deleteLocked()
}

// Pending OAuth requests live only for the lifetime of the login process.
func (s *insecureFileStore) GetAuthRequestInfo(_ context.Context, state string) (*oauth.AuthRequestData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil || s.pending.State != state {
		return nil, keyring.ErrNotFound
	}
	info := *s.pending
	return &info, nil
}

func (s *insecureFileStore) SaveAuthRequestInfo(_ context.Context, info oauth.AuthRequestData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = &info
	return nil
}

func (s *insecureFileStore) DeleteAuthRequestInfo(_ context.Context, state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != nil && s.pending.State == state {
		s.pending = nil
	}
	return nil
}

func (s *insecureFileStore) finishOAuthLogin(handle, callbackURL, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.loadLocked()
	if err != nil {
		return err
	}
	record.Handle = handle
	record.CallbackURL = callbackURL
	record.ClientID = clientID
	record.Active = true
	return s.saveLocked(*record)
}

func (s *insecureFileStore) cancelLogin() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = nil
}
