package app

import (
	"context"
	"fmt"
)

// AuthAccounts lists all stored accounts, marking the active one.
func (s *Service) AuthAccounts(ctx context.Context) ([]AuthAccountResult, error) {
	accounts, activeDID, err := s.auth.Accounts()
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	results := make([]AuthAccountResult, 0, len(accounts))
	for _, account := range accounts {
		handle := account.Handle
		resolved := s.resolveAuthor(ctx, account.DID)
		if resolved.Handle != account.DID {
			handle = resolved.Handle
		}
		results = append(results, AuthAccountResult{
			Active: account.DID == activeDID,
			DID:    account.DID, Handle: handle, Method: account.Method,
		})
	}
	return results, nil
}
