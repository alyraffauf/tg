package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/tangled"
)

// ListOptions filters and limits issue and pull-request listings.
type ListOptions struct {
	Author string
	State  string
	Limit  int64
	Order  string
}

// ListIssues lists issues in the target repository.
func (s *Service) ListIssues(ctx context.Context, t Target, options ListOptions) ([]Item, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}
	listOptions, err := s.issueListOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	issues, err := s.appview.ListIssues(ctx, repoDid, listOptions)
	if err != nil {
		return nil, fmt.Errorf("list issues for %q: %w", t.Repo, err)
	}
	return s.buildItems(ctx, issues.Items, decodeIssue), nil
}

func (s *Service) issueListOptions(ctx context.Context, options ListOptions) (tangled.ListOpts, error) {
	return s.appviewListOptions(ctx, options, "issue", "open", "closed")
}

func (s *Service) pullListOptions(ctx context.Context, options ListOptions) (tangled.ListOpts, error) {
	return s.appviewListOptions(ctx, options, "pull request", "open", "closed", "merged")
}

func (s *Service) appviewListOptions(ctx context.Context, options ListOptions, resource string, states ...string) (tangled.ListOpts, error) {
	if options.Limit < 0 {
		return tangled.ListOpts{}, fmt.Errorf("%s list limit must be positive", resource)
	}
	if options.State != "" && !isAllowedValue(options.State, states) {
		return tangled.ListOpts{}, fmt.Errorf("invalid %s state %q", resource, options.State)
	}
	if options.Order != "" && !isAllowedValue(options.Order, []string{"asc", "desc"}) {
		return tangled.ListOpts{}, fmt.Errorf("invalid list order %q", options.Order)
	}

	author, err := s.resolveAuthorFilter(ctx, options.Author)
	if err != nil {
		return tangled.ListOpts{}, err
	}
	return tangled.ListOpts{
		Author:   author,
		State:    options.State,
		Limit:    defaultListLimit,
		MaxItems: options.Limit,
		Order:    options.Order,
	}, nil
}

func (s *Service) resolveAuthorFilter(ctx context.Context, author string) (string, error) {
	if author == "" || strings.HasPrefix(author, "did:") {
		return author, nil
	}
	identity, err := s.resolver.ResolveHandle(ctx, author)
	if err != nil {
		return "", fmt.Errorf("resolve author %q: %w", author, err)
	}
	return identity.DID.String(), nil
}

func isAllowedValue(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
