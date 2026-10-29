package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/alyraffauf/tg/tangled"
)

// defaultListLimit is the page size used when listing issues, pulls, and
// records via the appview or PDS.
const defaultListLimit int64 = 100

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

// buildItems decodes listing records and reports records it could not use.
func (s *Service) buildItems(ctx context.Context, items []tangled.ListItem, pull bool) ([]Item, []RecordWarning) {
	result := make([]Item, 0, len(items))
	var warnings []RecordWarning

	for _, listItem := range items {
		ticket, err := decodeTicket(listItem.Value, pull)
		if err != nil {
			warnings = append(warnings, RecordWarning{URI: listItem.URI, Error: fmt.Sprintf("decode record: %v", err)})
			continue
		}

		updated := listItem.StateUpdatedAt
		if updated == "" {
			updated = ticket.CreatedAt
		}

		sourceBranch := ""
		if pull {
			patches, err := s.appview.ListPatches(ctx, listItem.URI)
			if err != nil {
				warnings = append(warnings, RecordWarning{URI: listItem.URI, Error: fmt.Sprintf("read patch source: %v", err)})
			} else if len(patches) > 0 && patches[len(patches)-1].Value.Source != nil {
				sourceBranch = patches[len(patches)-1].Value.Source.Branch
			}
		}

		title := ticket.Title.Text
		if title == "" {
			title = "(no title)"
		}

		result = append(result, Item{
			Rkey:         extractRKey(listItem.URI),
			URI:          listItem.URI,
			Title:        title,
			State:        listItem.State,
			Author:       s.resolveAuthor(ctx, listItem.Author),
			CreatedAt:    ticket.CreatedAt,
			UpdatedAt:    updated,
			CommentCount: listItem.CommentCount,
			SourceBranch: sourceBranch,
			TargetBranch: ticket.TargetBranch,
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
