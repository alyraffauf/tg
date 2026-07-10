package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/alyraffauf/tg/tangled"
)

const defaultListLimit int64 = 100

// shortDate trims an ISO 8601 timestamp to its YYYY-MM-DD prefix.
func shortDate(timestamp string) string {
	if len(timestamp) > 10 {
		return timestamp[:10]
	}
	return timestamp
}

// renderTable writes a tab-aligned table of rows to stdout under header.
// emptyMessage is shown when rows has no entries. Every renderer in this
// package (issues, pulls, repos, SSH keys) goes through this.
func renderTable(header []string, rows [][]string, emptyMessage string) {
	if len(rows) == 0 {
		fmt.Println(emptyMessage)
		return
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
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

// resolveAuthor resolves a DID to an author, falling back to the raw
// DID string for Handle if resolution fails.
func resolveAuthor(ctx context.Context, did string) author {
	result := author{DID: did}
	if ident, err := resolver.ResolveDID(ctx, did); err == nil {
		result.Handle = ident.Handle.String()
	} else {
		result.Handle = did
	}
	return result
}

// recordView is the fields common to an issue or pull-request record,
// as decoded from a tangled.ListItem's raw Value.
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

// buildItems decodes a listing's items into display/JSON-ready items,
// silently skipping any whose Value fails to decode. decode is
// decodeIssue or decodePull depending on the resource being listed.
func buildItems(ctx context.Context, items []tangled.ListItem, decode func(json.RawMessage) (recordView, error)) []item {
	result := make([]item, 0, len(items))

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

		result = append(result, item{
			Rkey:         extractRKey(listItem.URI),
			URI:          listItem.URI,
			Title:        title,
			State:        listItem.State,
			Author:       resolveAuthor(ctx, extractDID(listItem.URI)),
			CreatedAt:    decoded.CreatedAt,
			UpdatedAt:    updated,
			CommentCount: listItem.CommentCount,
			SourceBranch: decoded.SourceBranch,
			TargetBranch: decoded.TargetBranch,
		})
	}

	return result
}

// renderList renders issue or pull-request items as a table.
func renderList(items []item, emptyMessage string) {
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		rows = append(rows, []string{it.Rkey, it.Title, it.State, it.Author.Handle, shortDate(it.UpdatedAt)})
	}
	renderTable([]string{"RKEY", "TITLE", "STATE", "AUTHOR", "UPDATED"}, rows, emptyMessage)
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
