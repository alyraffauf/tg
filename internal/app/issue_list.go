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
	// Labels keeps items carrying every named label. Names match label
	// definitions exactly.
	Labels []string
}

// ListIssues lists issues in the target repository.
func (s *Service) ListIssues(ctx context.Context, t Target, options ListOptions) (*ItemListResult, error) {
	listOptions, err := s.issueListOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	return s.listTickets(ctx, t, options, listOptions, false)
}

// listTickets lists issues or pulls and applies label filters. The appview
// cannot filter by label, so tg filters each page as it arrives and keeps
// paging until the caller's limit is met or the list ends.
func (s *Service) listTickets(ctx context.Context, t Target, options ListOptions, listOptions tangled.ListOpts, pull bool) (*ItemListResult, error) {
	repoDid, err := s.repoDID(ctx, t)
	if err != nil {
		return nil, err
	}

	var catalog labelCatalog
	var warnings []RecordWarning
	if len(options.Labels) > 0 {
		catalog, err = s.loadCompleteLabelCatalog(ctx, repoDid)
		if err != nil {
			return nil, fmt.Errorf("read label definitions for %q: %w", t.Repo, err)
		}
		requiredKeySets, err := catalog.filterKeySets(options.Labels)
		if err != nil {
			return nil, err
		}
		listOptions.Match = func(ticket tangled.Ticket) bool {
			return matchesAllLabels(ticket.TangledLabels, requiredKeySets)
		}
	}

	list, err := s.listAppviewTickets(ctx, repoDid, listOptions, pull)
	if err != nil {
		return nil, fmt.Errorf("list %s for %q: %w", ticketKindPlural(pull), t.Repo, err)
	}
	if len(options.Labels) == 0 {
		catalog = newLabelCatalog(repoDid, nil)
		for _, item := range list.Items {
			ticket, err := decodeTicket(item.Value, pull)
			if err == nil && len(appliedLabelKeys(ticket.TangledLabels)) > 0 {
				catalog, warnings = s.loadDisplayLabelCatalog(ctx, repoDid)
				break
			}
		}
	}
	warnings = append(warnings, recordWarnings(list.Warnings)...)

	items, itemWarnings := s.buildItems(ctx, list.Items, pull, catalog)
	return &ItemListResult{Items: items, Warnings: append(warnings, itemWarnings...)}, nil
}

func (s *Service) listAppviewTickets(ctx context.Context, repoDid string, listOptions tangled.ListOpts, pull bool) (*tangled.List, error) {
	if pull {
		return s.appview.ListPulls(ctx, repoDid, listOptions)
	}
	return s.appview.ListIssues(ctx, repoDid, listOptions)
}

func ticketKindPlural(pull bool) string {
	if pull {
		return "PRs"
	}
	return "issues"
}

func (s *Service) issueListOptions(ctx context.Context, options ListOptions) (tangled.ListOpts, error) {
	return s.appviewListOptions(ctx, options, "issue", "open", "closed", "completed")
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
