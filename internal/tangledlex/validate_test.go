package tangledlex

import (
	"strings"
	"testing"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"
)

const (
	testDID   = "did:plc:abc123"
	testATURI = "at://did:plc:abc123/sh.tangled.repo.issue/3k2abc"
	testPull  = "at://did:plc:abc123/sh.tangled.repo.pull/3k2abc"
	testCID   = "bafybeigdyrzt5m6b5nkn55vsgzzfw5cfs2tidw6zqugycdkyybf2z7kz4q"
	testTime  = "2026-07-25T12:00:00Z"
)

func TestValidateRecordAcceptsEveryWritableRecord(t *testing.T) {
	for collection, record := range validRecords() {
		t.Run(collection, func(t *testing.T) {
			if err := ValidateRecord(collection, record); err != nil {
				t.Fatalf("ValidateRecord() error = %v", err)
			}
		})
	}
}

func TestValidateRecordAcceptsPreservedRepoMap(t *testing.T) {
	record := map[string]any{
		"$type":     "sh.tangled.repo",
		"knot":      "knot.example",
		"createdAt": testTime,
		"custom":    map[string]any{"preserved": true},
	}
	if err := ValidateRecord("sh.tangled.repo", record); err != nil {
		t.Fatalf("ValidateRecord() error = %v", err)
	}
}

func TestValidateRecordRejectsEveryWritableRecord(t *testing.T) {
	tests := []struct {
		name       string
		collection string
		record     any
		contains   string
	}{
		{"repo", "sh.tangled.repo", Repo{LexiconTypeID: "sh.tangled.repo", CreatedAt: testTime}, "knot"},
		{"issue", "sh.tangled.repo.issue", RepoIssue{LexiconTypeID: "sh.tangled.repo.issue", Title: "title", CreatedAt: testTime}, "repo"},
		{"feed comment", "sh.tangled.feed.comment", FeedComment{LexiconTypeID: "sh.tangled.feed.comment", Body: &FeedComment_Body{MarkupMarkdown: &MarkupMarkdown{Text: "body"}}, CreatedAt: testTime}, "subject"},
		{"issue state", "sh.tangled.repo.issue.state", RepoIssueState{LexiconTypeID: "sh.tangled.repo.issue.state", Issue: testATURI, CreatedAt: testTime}, "state"},
		{"pull", "sh.tangled.repo.pull", RepoPull{LexiconTypeID: "sh.tangled.repo.pull", Title: "title", CreatedAt: testTime}, "target"},
		{"pull status", "sh.tangled.repo.pull.status", RepoPullStatus{LexiconTypeID: "sh.tangled.repo.pull.status", Pull: testATURI, CreatedAt: testTime}, "status"},
		{"public key", "sh.tangled.publicKey", PublicKey{LexiconTypeID: "sh.tangled.publicKey", Name: "key", CreatedAt: testTime}, "key"},
		{"string", "sh.tangled.string", String{LexiconTypeID: "sh.tangled.string", Contents: "text", CreatedAt: testTime}, "filename"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateRecord(test.collection, test.record)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("ValidateRecord() error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestValidateRecordRejectsMismatchedType(t *testing.T) {
	err := ValidateRecord("sh.tangled.repo.issue", RepoIssue{
		LexiconTypeID: "sh.tangled.repo.pull",
		Repo:          testDID,
		Title:         "title",
		CreatedAt:     testTime,
	})
	if err == nil || !strings.Contains(err.Error(), "$type") {
		t.Fatalf("ValidateRecord() error = %v, want type error", err)
	}
}

func TestValidateRecordRejectsNestedConstraints(t *testing.T) {
	tests := []struct {
		name       string
		collection string
		record     func() any
		contains   string
	}{
		{
			name:       "repo website",
			collection: "sh.tangled.repo",
			record: func() any {
				record := validRecords()["sh.tangled.repo"].(Repo)
				record.Website = stringPointer(":not-a-uri")
				return record
			},
			contains: "website",
		},
		{
			name:       "issue mention",
			collection: "sh.tangled.repo.issue",
			record: func() any {
				record := validRecords()["sh.tangled.repo.issue"].(RepoIssue)
				record.Mentions = []string{"not-a-did"}
				return record
			},
			contains: "mentions",
		},
		{
			name:       "feed comment subject URI",
			collection: "sh.tangled.feed.comment",
			record: func() any {
				record := validRecords()["sh.tangled.feed.comment"].(FeedComment)
				record.Subject.Uri = "not-an-at-uri"
				return record
			},
			contains: "subject.uri",
		},
		{
			name:       "feed comment subject CID",
			collection: "sh.tangled.feed.comment",
			record: func() any {
				record := validRecords()["sh.tangled.feed.comment"].(FeedComment)
				record.Subject.Cid = "not-a-cid"
				return record
			},
			contains: "subject.cid",
		},
		{
			name:       "feed comment pull round",
			collection: "sh.tangled.feed.comment",
			record: func() any {
				record := validRecords()["sh.tangled.feed.comment"].(FeedComment)
				record.Subject.Uri = testPull
				record.PullRoundIdx = nil
				return record
			},
			contains: "pullRoundIdx",
		},
		{
			name:       "feed comment pull round index",
			collection: "sh.tangled.feed.comment",
			record: func() any {
				record := validRecords()["sh.tangled.feed.comment"].(FeedComment)
				index := int64(-1)
				record.PullRoundIdx = &index
				return record
			},
			contains: "non-negative",
		},
		{
			name:       "feed comment body text",
			collection: "sh.tangled.feed.comment",
			record: func() any {
				record := validRecords()["sh.tangled.feed.comment"].(FeedComment)
				record.Body.MarkupMarkdown.Text = ""
				return record
			},
			contains: "body.text",
		},
		{
			name:       "pull blob reference",
			collection: "sh.tangled.repo.pull",
			record: func() any {
				record := validRecords()["sh.tangled.repo.pull"].(RepoPull)
				record.Rounds[0].PatchBlob.Ref = lexutil.LexLink{}
				return record
			},
			contains: "reference",
		},
		{
			name:       "pull source repository",
			collection: "sh.tangled.repo.pull",
			record: func() any {
				record := validRecords()["sh.tangled.repo.pull"].(RepoPull)
				record.Source = &RepoPull_Source{Branch: "main", Repo: stringPointer("not-a-did")}
				return record
			},
			contains: "source.repo",
		},
		{
			name:       "pull reference",
			collection: "sh.tangled.repo.pull",
			record: func() any {
				record := validRecords()["sh.tangled.repo.pull"].(RepoPull)
				record.DependentOn = stringPointer("not-an-at-uri")
				return record
			},
			contains: "dependentOn",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateRecord(test.collection, test.record())
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("ValidateRecord() error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestValidateRecordUsesGraphemeAndByteLimits(t *testing.T) {
	family := "👨‍👩‍👧‍👦"
	repo := validRecords()["sh.tangled.repo"].(Repo)
	repo.Description = stringPointer(strings.Repeat(family, 140))
	if err := ValidateRecord("sh.tangled.repo", repo); err != nil {
		t.Fatalf("140 graphemes rejected: %v", err)
	}
	repo.Description = stringPointer(strings.Repeat(family, 141))
	if err := ValidateRecord("sh.tangled.repo", repo); err == nil {
		t.Fatal("141 graphemes accepted")
	}

	key := validRecords()["sh.tangled.publicKey"].(PublicKey)
	key.Key = strings.Repeat("é", 2049)
	if err := ValidateRecord("sh.tangled.publicKey", key); err == nil {
		t.Fatal("public key over 4096 UTF-8 bytes accepted")
	}
}

func TestValidateRecordAcceptsFutureKnownValues(t *testing.T) {
	issueState := validRecords()["sh.tangled.repo.issue.state"].(RepoIssueState)
	issueState.State = "sh.tangled.repo.issue.state.archived"
	if err := ValidateRecord("sh.tangled.repo.issue.state", issueState); err != nil {
		t.Fatalf("future issue state rejected: %v", err)
	}
	pullStatus := validRecords()["sh.tangled.repo.pull.status"].(RepoPullStatus)
	pullStatus.Status = "sh.tangled.repo.pull.status.draft"
	if err := ValidateRecord("sh.tangled.repo.pull.status", pullStatus); err != nil {
		t.Fatalf("future pull status rejected: %v", err)
	}
}

func TestValidateMarkdownBlobConstraints(t *testing.T) {
	record := validRecords()["sh.tangled.feed.comment"].(FeedComment)
	validLink := lexutil.LexLink(cid.MustParse(testCID))
	tests := []struct {
		name string
		blob *lexutil.LexBlob
	}{
		{name: "nil", blob: nil},
		{name: "missing reference", blob: &lexutil.LexBlob{MimeType: "image/png"}},
		{name: "non-image", blob: &lexutil.LexBlob{Ref: validLink, MimeType: "text/plain"}},
		{name: "negative size", blob: &lexutil.LexBlob{Ref: validLink, MimeType: "image/png", Size: -1}},
		{name: "oversized", blob: &lexutil.LexBlob{Ref: validLink, MimeType: "image/png", Size: 1_000_001}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := record
			markdown := *record.Body.MarkupMarkdown
			markdown.Blobs = []*lexutil.LexBlob{test.blob}
			candidate.Body = &FeedComment_Body{MarkupMarkdown: &markdown}
			if err := ValidateRecord("sh.tangled.feed.comment", candidate); err == nil {
				t.Fatal("ValidateRecord() accepted invalid markdown blob")
			}
		})
	}
}

func validRecords() map[string]any {
	return map[string]any{
		"sh.tangled.repo":       Repo{LexiconTypeID: "sh.tangled.repo", Knot: "knot.example", CreatedAt: testTime},
		"sh.tangled.repo.issue": RepoIssue{LexiconTypeID: "sh.tangled.repo.issue", Repo: testDID, Title: "title", CreatedAt: testTime},
		"sh.tangled.feed.comment": FeedComment{
			LexiconTypeID: "sh.tangled.feed.comment",
			Subject:       &comatproto.RepoStrongRef{Uri: testATURI, Cid: testCID},
			Body:          &FeedComment_Body{MarkupMarkdown: &MarkupMarkdown{Text: "body"}},
			CreatedAt:     testTime,
		},
		"sh.tangled.repo.issue.state": RepoIssueState{LexiconTypeID: "sh.tangled.repo.issue.state", Issue: testATURI, State: "sh.tangled.repo.issue.state.open", CreatedAt: testTime},
		"sh.tangled.repo.pull": RepoPull{
			LexiconTypeID: "sh.tangled.repo.pull",
			Title:         "title",
			CreatedAt:     testTime,
			Target:        &RepoPull_Target{Repo: testDID, Branch: "main"},
			Rounds: []*RepoPull_Round{{
				CreatedAt: testTime,
				PatchBlob: &lexutil.LexBlob{Ref: lexutil.LexLink(cid.MustParse("bafybeigdyrzt5m6b5nkn55vsgzzfw5cfs2tidw6zqugycdkyybf2z7kz4q")), MimeType: "application/gzip"},
			}},
		},
		"sh.tangled.repo.pull.status": RepoPullStatus{LexiconTypeID: "sh.tangled.repo.pull.status", Pull: testATURI, Status: "sh.tangled.repo.pull.status.open", CreatedAt: testTime},
		"sh.tangled.publicKey":        PublicKey{LexiconTypeID: "sh.tangled.publicKey", Key: "ssh-ed25519 AAAA", Name: "key", CreatedAt: testTime},
		"sh.tangled.string":           String{LexiconTypeID: "sh.tangled.string", Filename: "note.md", Description: "note", Contents: "text", CreatedAt: testTime},
	}
}

func stringPointer(value string) *string {
	return &value
}
