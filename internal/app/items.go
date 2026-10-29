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
	var ticket tangled.Ticket
	if err := json.Unmarshal(raw, &ticket); err != nil {
		return recordView{}, err
	}
	if ticket.Type != tangled.IssueCollection {
		return recordView{}, fmt.Errorf("unexpected ticket type %q", ticket.Type)
	}
	body := ""
	if ticket.Body != nil {
		body = ticket.Body.Text
	}
	return recordView{Title: ticket.Title.Text, Body: body, CreatedAt: ticket.CreatedAt, TargetBranch: ticket.TargetBranch}, nil
}
func decodePull(raw json.RawMessage) (recordView, error) { return decodeIssue(raw) }

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// resolveAuthor uses the DID when an author has no usable handle.
func (s *Service) resolveAuthor(ctx context.Context, did string) Author {
	return Author{DID: did, Handle: s.ownerHandle(ctx, did)}
}

// buildItems decodes a listing's items into display/JSON-ready items,
// silently skipping any whose Value fails to decode. decode is decodeIssue
// or decodePull depending on the resource being listed.
func (s *Service) buildItems(ctx context.Context, items []tangled.ListItem, decode func(json.RawMessage) (recordView, error)) ([]Item, []RecordWarning) {
	result := make([]Item, 0, len(items))
	var warnings []RecordWarning

	for _, listItem := range items {
		decoded, err := decode(listItem.Value)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: listItem.URI, Error: fmt.Sprintf("decode record: %v", err)})
			continue
		}

		updated := listItem.StateUpdatedAt
		if updated == "" {
			updated = decoded.CreatedAt
		}

		if decoded.TargetBranch != "" {
			patches, err := s.appview.ListPatches(ctx, listItem.URI)
			if err != nil {
				warnings = append(warnings, RecordWarning{URI: listItem.URI, Error: fmt.Sprintf("read patch source: %v", err)})
			} else if len(patches) > 0 && patches[len(patches)-1].Value.Source != nil {
				decoded.SourceBranch = patches[len(patches)-1].Value.Source.Branch
			}
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
			Author:       s.resolveAuthor(ctx, listItem.Author),
			CreatedAt:    decoded.CreatedAt,
			UpdatedAt:    updated,
			CommentCount: listItem.CommentCount,
			SourceBranch: decoded.SourceBranch,
			TargetBranch: decoded.TargetBranch,
		})
	}

	return result, warnings
}

func recordWarnings(warnings []tangled.RecordWarning) []RecordWarning {
	result := make([]RecordWarning, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, RecordWarning{URI: warning.URI, Error: warning.Error})
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
