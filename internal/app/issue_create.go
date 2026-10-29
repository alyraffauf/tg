package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/tangled"
)

func (s *Service) CreateIssue(ctx context.Context, t Target, title, body string) (*CreatedRecordResult, error) {
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("title must not be empty")
	}
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	if err := s.requireNative(ctx, repo); err != nil {
		return nil, err
	}
	receipt, warnings, err := s.socialProcedure(ctx, repo, "com.atproto.repo.createRecord", map[string]any{"collection": tangled.IssueCollection, "record": ticketRecord(title, body, "")})

	if err != nil {
		return nil, err
	}
	return &CreatedRecordResult{URI: receipt.URI, CID: receipt.CID, Rkey: extractRKey(receipt.URI), Warnings: warnings}, nil
}
