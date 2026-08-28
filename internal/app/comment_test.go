package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
)

const commentSubjectCID = "bafybeigdyrzt5m6b5nkn55vsgzzfw5cfs2tidw6zqugycdkyybf2z7kz4q"

func TestCommentIssueWritesFeedComment(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	issue := tangled.ListItem{
		URI: "at://did:plc:owner/sh.tangled.repo.issue/issue-1",
		CID: commentSubjectCID,
	}
	service.appview = testAppview{
		repo:   commentTestRepo(),
		issues: &tangled.List{Items: []tangled.ListItem{issue}},
	}

	if _, err := service.CommentIssue(context.Background(), commentTestTarget(), "issue-1", "comment body"); err != nil {
		t.Fatalf("CommentIssue() error = %v", err)
	}
	if pds.getCalls != 0 {
		t.Fatalf("PDS GetRecord calls = %d, want 0", pds.getCalls)
	}
	assertFeedCommentWrite(t, pds, issue, "comment body", nil)
}

func TestCommentPullWritesFeedCommentForLatestRound(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	pullValue, err := json.Marshal(tangledlex.RepoPull{Rounds: []*tangledlex.RepoPull_Round{{}, {}}})
	if err != nil {
		t.Fatalf("marshal pull: %v", err)
	}
	pull := tangled.ListItem{
		URI:   "at://did:plc:owner/sh.tangled.repo.pull/pull-1",
		CID:   commentSubjectCID,
		Value: pullValue,
	}
	service.appview = testAppview{
		repo:  commentTestRepo(),
		pulls: &tangled.List{Items: []tangled.ListItem{pull}},
	}

	if _, err := service.CommentPull(context.Background(), commentTestTarget(), "pull-1", "comment body"); err != nil {
		t.Fatalf("CommentPull() error = %v", err)
	}
	if pds.getCalls != 0 {
		t.Fatalf("PDS GetRecord calls = %d, want 0", pds.getCalls)
	}
	latestRoundIdx := int64(1)
	assertFeedCommentWrite(t, pds, pull, "comment body", &latestRoundIdx)
}

func TestCommentPullRejectsPullWithoutRounds(t *testing.T) {
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	pullValue, err := json.Marshal(tangledlex.RepoPull{})
	if err != nil {
		t.Fatalf("marshal pull: %v", err)
	}
	service.appview = testAppview{
		repo: commentTestRepo(),
		pulls: &tangled.List{Items: []tangled.ListItem{{
			URI:   "at://did:plc:owner/sh.tangled.repo.pull/pull-1",
			CID:   commentSubjectCID,
			Value: pullValue,
		}}},
	}

	_, err = service.CommentPull(context.Background(), commentTestTarget(), "pull-1", "comment body")
	if err == nil || !strings.Contains(err.Error(), "has no rounds") {
		t.Fatalf("CommentPull() error = %v, want no-rounds error", err)
	}
	if len(pds.puts) != 0 {
		t.Fatalf("PDS writes = %d, want 0", len(pds.puts))
	}
}

func TestCreateFeedCommentRejectsInvalidSubjectCID(t *testing.T) {
	for _, subjectCID := range []string{"", "not-a-cid"} {
		t.Run(subjectCID, func(t *testing.T) {
			pds := &testPDS{}
			service := testService(pds, &testGit{}, &testKnot{})

			_, err := service.createFeedComment(context.Background(), tangled.ListItem{
				URI: "at://did:plc:owner/sh.tangled.repo.issue/issue-1",
				CID: subjectCID,
			}, "comment body", nil)
			if err == nil || !strings.Contains(err.Error(), "subject CID") {
				t.Fatalf("createFeedComment() error = %v, want CID error", err)
			}
			if len(pds.puts) != 0 {
				t.Fatalf("PDS writes = %d, want 0", len(pds.puts))
			}
		})
	}
}

func assertFeedCommentWrite(t *testing.T, pds *testPDS, subject tangled.ListItem, body string, pullRoundIdx *int64) {
	t.Helper()
	if len(pds.puts) != 1 {
		t.Fatalf("PDS writes = %d, want 1", len(pds.puts))
	}
	put := pds.puts[0]
	if put.Collection != tangled.FeedCommentCollection {
		t.Fatalf("collection = %q, want %q", put.Collection, tangled.FeedCommentCollection)
	}
	record, ok := put.Record.(tangledlex.FeedComment)
	if !ok {
		t.Fatalf("record type = %T, want FeedComment", put.Record)
	}
	if record.LexiconTypeID != tangled.FeedCommentCollection {
		t.Errorf("record $type = %q, want %q", record.LexiconTypeID, tangled.FeedCommentCollection)
	}
	if record.Subject == nil || record.Subject.Uri != subject.URI || record.Subject.Cid != subject.CID {
		t.Errorf("record subject = %+v, want URI %q and CID %q", record.Subject, subject.URI, subject.CID)
	}
	if record.Body == nil || record.Body.MarkupMarkdown == nil || record.Body.MarkupMarkdown.Text != body || record.Body.MarkupMarkdown.Original == nil || *record.Body.MarkupMarkdown.Original != body {
		t.Errorf("record body = %+v, want markdown %q", record.Body, body)
	}
	if !sameRoundIndex(record.PullRoundIdx, pullRoundIdx) {
		t.Errorf("pullRoundIdx = %v, want %v", record.PullRoundIdx, pullRoundIdx)
	}
}

func sameRoundIndex(left, right *int64) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func commentTestRepo() *tangled.Repo {
	return &tangled.Repo{
		URI: "at://did:plc:owner/sh.tangled.repo/example",
		Value: tangledlex.Repo{
			Knot:    "knot.example",
			RepoDid: optionalString("did:plc:repository"),
		},
	}
}

func commentTestTarget() Target {
	return Target{Handle: "owner.test", Repo: "example"}
}
