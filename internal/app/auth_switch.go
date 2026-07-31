package app

import (
	"context"
	"fmt"
)

// SwitchAccount selects the active account by handle or DID.
func (s *Service) SwitchAccount(ctx context.Context, selector string) (*AuthAccountResult, error) {
	account, err := s.auth.SelectAccount(selector)
	if err != nil {
		return nil, fmt.Errorf("select account %q: %w", selector, err)
	}
	resolved := s.resolveAuthor(ctx, account.DID)
	return &AuthAccountResult{
		Active: true, DID: account.DID, Handle: resolved.Handle, Method: account.Method,
	}, nil
}
