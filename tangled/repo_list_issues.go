package tangled

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (t *Tangled) ListIssues(ctx context.Context, repo string, opts ListOpts) (*List, error) {
	return t.listTickets(ctx, repo, opts, false)
}
func (t *Tangled) listTickets(ctx context.Context, repo string, opts ListOpts, pulls bool) (*List, error) {
	method := "org.tangled.repo.listIssues"
	if pulls {
		method = "org.tangled.repo.listPulls"
	}
	var warnings []RecordWarning
	items, err := fetchAllPages(ctx, opts.MaxItems, func(ctx context.Context, cursor string) ([]ListItem, *string, error) {
		params := opts.params(repo, cursor)
		delete(params, "author")
		if pulls {
			delete(params, "state")
			if opts.State != "" {
				state := opts.State
				if state == "merged" {
					state = "completed"
				}
				params["status"] = state
			}
		}
		var page struct {
			Items  []json.RawMessage `json:"items"`
			Cursor *string           `json:"cursor"`
		}
		if err := t.Client.Get(ctx, syntax.NSID(method), params, &page); err != nil {
			return nil, nil, err
		}
		var decoded []ListItem
		for _, raw := range page.Items {
			var item ListItem
			if err := json.Unmarshal(raw, &item); err != nil {
				warnings = append(warnings, RecordWarning{Error: err.Error()})
				continue
			}
			var ticket Ticket
			if err := json.Unmarshal(item.Value, &ticket); err != nil || ticket.Type != IssueCollection {
				warnings = append(warnings, RecordWarning{URI: item.URI, Error: "invalid ticket record"})
				continue
			}
			if opts.Author != "" && item.Author != opts.Author {
				continue
			}
			if pulls && item.State == "completed" {
				item.State = "merged"
			}
			decoded = append(decoded, item)
		}
		return decoded, page.Cursor, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	return &List{Items: items, Warnings: warnings}, nil
}
func (t *Tangled) GetTicket(ctx context.Context, uri string) (*ListItem, error) {
	var value struct {
		URI    string `json:"uri"`
		CID    string `json:"cid"`
		Author struct {
			DID string `json:"did"`
		} `json:"author"`
		Title        Markdown  `json:"title"`
		Body         *Markdown `json:"body"`
		State        string    `json:"state"`
		TargetBranch string    `json:"targetBranch"`
		CreatedAt    string    `json:"createdAt"`
	}
	if err := t.Client.Get(ctx, syntax.NSID("org.tangled.track.getTicketView"), map[string]any{"uri": uri}, &value); err != nil {
		return nil, err
	}
	ticket := Ticket{Type: IssueCollection, Title: value.Title, Body: value.Body, State: value.State, TargetBranch: value.TargetBranch, CreatedAt: value.CreatedAt}
	data, err := json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	return &ListItem{URI: value.URI, CID: value.CID, Value: data, State: value.State, Author: value.Author.DID}, nil
}
func (t *Tangled) ListPatches(ctx context.Context, ticket string) ([]Patch, error) {
	return fetchAllPages(ctx, 0, func(ctx context.Context, cursor string) ([]Patch, *string, error) {
		var page struct {
			Items  []Patch `json:"items"`
			Cursor *string `json:"cursor"`
		}
		params := map[string]any{"ticket": ticket, "order": "asc", "limit": 500}
		if cursor != "" {
			params["cursor"] = cursor
		}
		err := t.Client.Get(ctx, syntax.NSID("org.tangled.temp.review.listPatches"), params, &page)
		return page.Items, page.Cursor, err
	})
}
