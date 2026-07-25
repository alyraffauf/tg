package tangledlex

import (
	"strings"
	"testing"

	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"
)

const (
	testDID   = "did:plc:abc123"
	testATURI = "at://did:plc:abc123/sh.tangled.repo.issue/3k2abc"
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
		{"issue comment", "sh.tangled.repo.issue.comment", RepoIssueComment{LexiconTypeID: "sh.tangled.repo.issue.comment", Body: "body", CreatedAt: testTime}, "issue"},
		{"issue state", "sh.tangled.repo.issue.state", RepoIssueState{LexiconTypeID: "sh.tangled.repo.issue.state", Issue: testATURI, State: "invalid", CreatedAt: testTime}, "state"},
		{"pull", "sh.tangled.repo.pull", RepoPull{LexiconTypeID: "sh.tangled.repo.pull", Title: "title", CreatedAt: testTime}, "target"},
		{"pull comment", "sh.tangled.repo.pull.comment", RepoPullComment{LexiconTypeID: "sh.tangled.repo.pull.comment", Body: "body", CreatedAt: testTime}, "pull"},
		{"pull status", "sh.tangled.repo.pull.status", RepoPullStatus{LexiconTypeID: "sh.tangled.repo.pull.status", Pull: testATURI, Status: "invalid", CreatedAt: testTime}, "status"},
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
			name:       "issue comment reply",
			collection: "sh.tangled.repo.issue.comment",
			record: func() any {
				record := validRecords()["sh.tangled.repo.issue.comment"].(RepoIssueComment)
				record.ReplyTo = stringPointer("not-an-at-uri")
				return record
			},
			contains: "replyTo",
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
		{
			name:       "pull comment reference",
			collection: "sh.tangled.repo.pull.comment",
			record: func() any {
				record := validRecords()["sh.tangled.repo.pull.comment"].(RepoPullComment)
				record.References = []string{"not-an-at-uri"}
				return record
			},
			contains: "references",
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

func validRecords() map[string]any {
	return map[string]any{
		"sh.tangled.repo":               Repo{LexiconTypeID: "sh.tangled.repo", Knot: "knot.example", CreatedAt: testTime},
		"sh.tangled.repo.issue":         RepoIssue{LexiconTypeID: "sh.tangled.repo.issue", Repo: testDID, Title: "title", CreatedAt: testTime},
		"sh.tangled.repo.issue.comment": RepoIssueComment{LexiconTypeID: "sh.tangled.repo.issue.comment", Issue: testATURI, Body: "body", CreatedAt: testTime},
		"sh.tangled.repo.issue.state":   RepoIssueState{LexiconTypeID: "sh.tangled.repo.issue.state", Issue: testATURI, State: "sh.tangled.repo.issue.state.open", CreatedAt: testTime},
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
		"sh.tangled.repo.pull.comment": RepoPullComment{LexiconTypeID: "sh.tangled.repo.pull.comment", Pull: testATURI, Body: "body", CreatedAt: testTime},
		"sh.tangled.repo.pull.status":  RepoPullStatus{LexiconTypeID: "sh.tangled.repo.pull.status", Pull: testATURI, Status: "sh.tangled.repo.pull.status.open", CreatedAt: testTime},
		"sh.tangled.publicKey":         PublicKey{LexiconTypeID: "sh.tangled.publicKey", Key: "ssh-ed25519 AAAA", Name: "key", CreatedAt: testTime},
		"sh.tangled.string":            String{LexiconTypeID: "sh.tangled.string", Filename: "note.md", Description: "note", Contents: "text", CreatedAt: testTime},
	}
}

func stringPointer(value string) *string {
	return &value
}
