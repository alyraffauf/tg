package app

import (
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

func TestEditLexiconRecordKeepsValidIssueWritable(t *testing.T) {
	title := "Updated title"
	body := "Updated body"
	record, err := editLexiconRecord(issueCollection, map[string]any{
		"$type":     issueCollection,
		"repo":      "did:plc:abc123",
		"title":     "Original title",
		"createdAt": "2026-07-25T12:00:00Z",
	}, &title, &body)
	if err != nil {
		t.Fatalf("editLexiconRecord() error = %v", err)
	}

	issue, ok := record.(tangledlex.RepoIssue)
	if !ok {
		t.Fatalf("editLexiconRecord() record = %T, want tangledlex.RepoIssue", record)
	}
	if issue.Title != title || issue.Body == nil || *issue.Body != body {
		t.Fatalf("edited issue = %+v", issue)
	}
	if err := tangledlex.ValidateRecord(issueCollection, issue); err != nil {
		t.Fatalf("ValidateRecord() error = %v", err)
	}
}

func TestEditLexiconRecordRejectsHistoricalRecordAtWriteBoundary(t *testing.T) {
	title := "Updated title"
	record, err := editLexiconRecord(issueCollection, map[string]any{
		"repo":      "did:plc:abc123",
		"title":     "Original title",
		"createdAt": "2026-07-25T12:00:00Z",
	}, &title, nil)
	if err != nil {
		t.Fatalf("editLexiconRecord() error = %v", err)
	}
	if err := tangledlex.ValidateRecord(issueCollection, record); err == nil || !strings.Contains(err.Error(), "$type") {
		t.Fatalf("ValidateRecord() error = %v, want type validation error", err)
	}
}
