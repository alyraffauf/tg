package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/tangled"
)

// defaultListLimit is the page size used when listing issues, pulls, and
// records via the appview or PDS.
const defaultListLimit int64 = 100

// recordView holds the fields common to an issue or pull-request record,
// decoded from a tangled.ListItem's raw Value.
type recordView struct {
	Title        string
	Body         string
	CreatedAt    string
	SourceBranch string
	TargetBranch string
}

func decodeIssue(raw json.RawMessage) (recordView, error) {
	var r tangled.IssueRecord
	if err := json.Unmarshal(raw, &r); err != nil {
		return recordView{}, err
	}
	return recordView{Title: r.Title, Body: r.Body, CreatedAt: r.CreatedAt}, nil
}

func decodePull(raw json.RawMessage) (recordView, error) {
	var r tangled.PullRecord
	if err := json.Unmarshal(raw, &r); err != nil {
		return recordView{}, err
	}
	return recordView{
		Title:        r.Title,
		Body:         r.Body,
		CreatedAt:    r.CreatedAt,
		SourceBranch: r.Source.Branch,
		TargetBranch: r.Target.Branch,
	}, nil
}

// resolveAuthor resolves a DID to an author, falling back to the raw DID
// string for Handle if resolution fails.
func (s *Service) resolveAuthor(ctx context.Context, did string) Author {
	result := Author{DID: did}
	if ident, err := s.resolver.ResolveDID(ctx, did); err == nil {
		result.Handle = ident.Handle.String()
	} else {
		result.Handle = did
	}
	return result
}

// buildItems decodes a listing's items into display/JSON-ready items,
// silently skipping any whose Value fails to decode. decode is decodeIssue
// or decodePull depending on the resource being listed.
func (s *Service) buildItems(ctx context.Context, items []tangled.ListItem, decode func(json.RawMessage) (recordView, error)) []Item {
	result := make([]Item, 0, len(items))

	for _, listItem := range items {
		decoded, err := decode(listItem.Value)
		if err != nil {
			continue
		}

		updated := listItem.StateUpdatedAt
		if updated == "" {
			updated = decoded.CreatedAt
		}

		title := decoded.Title
		if title == "" {
			title = "(no title)"
		}

		result = append(result, Item{
			Rkey:         extractRKey(listItem.URI),
			URI:          listItem.URI,
			Title:        title,
			State:        listItem.State,
			Author:       s.resolveAuthor(ctx, extractDID(listItem.URI)),
			CreatedAt:    decoded.CreatedAt,
			UpdatedAt:    updated,
			CommentCount: listItem.CommentCount,
			SourceBranch: decoded.SourceBranch,
			TargetBranch: decoded.TargetBranch,
		})
	}

	return result
}

func extractDID(uri string) string {
	uri = strings.TrimPrefix(uri, "at://")
	did, _, _ := strings.Cut(uri, "/")
	return did
}

func extractRKey(uri string) string {
	if idx := strings.LastIndex(uri, "/"); idx != -1 {
		return uri[idx+1:]
	}
	return uri
}

// findByRKey finds the listing item whose URI ends in "/"+rkey. what names
// the resource kind (e.g. "issue", "pull request") for the error message.
func findByRKey(items []tangled.ListItem, rkey, what string) (*tangled.ListItem, error) {
	for i := range items {
		if strings.HasSuffix(items[i].URI, "/"+rkey) {
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("%s %q not found", what, rkey)
}
