package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"
)

func TestNewPullRecordUsesDistinctSourceAndTarget(t *testing.T) {
	record, err := newPullRecord(pullRecordInput{
		Title:         "Cross-repo change",
		TargetRepoDid: "did:plc:upstream",
		SourceRepoDid: "did:plc:fork",
		Base:          "main",
		Head:          "feature",
		Patch:         &atproto.Blob{},
	}, time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("newPullRecord() error = %v", err)
	}

	if record.Target.Repo != "did:plc:upstream" {
		t.Fatalf("unexpected target: %+v", record.Target)
	}
	if stringValue(record.Source.Repo) != "did:plc:fork" {
		t.Fatalf("unexpected source: %+v", record.Source)
	}
}

func TestCreatePullUsesBaseRevisionAndRecordsCanonicalBranch(t *testing.T) {
	patchCID := cid.MustParse("bafybeibwzifrf5tfwmbtw6ewjqr5q6rh5y5b6gzzledmxce5ilrjzsozoa")
	pds := &testPDS{uploadBlob: &atproto.Blob{Type: "blob", MimeType: "application/gzip", Ref: lexutil.LexLink(patchCID), Size: 42}}
	git := &testGit{
		patch: []byte("patch"),
		pullBases: map[string]gitutil.PullBase{
			"upstream/main": {Revision: "base-commit", Branch: "main"},
		},
	}
	service := testService(pds, git, &testKnot{})
	service.appview = testAppview{repos: map[string]*tangled.Repo{
		"at://did:plc:owner/sh.tangled.repo/target": {
			URI:   "at://did:plc:owner/sh.tangled.repo/target",
			Value: tangledlex.Repo{RepoDid: optionalString("did:plc:target-repo")},
		},
		"at://did:plc:owner/sh.tangled.repo/fork": {
			URI:   "at://did:plc:owner/sh.tangled.repo/fork",
			Value: tangledlex.Repo{RepoDid: optionalString("did:plc:fork-repo")},
		},
	}}
	source := Target{Handle: "owner.test", Repo: "fork"}

	result, err := service.CreatePull(context.Background(), CreatePullInput{
		RepoDir: "/tmp/repo",
		Title:   "Example",
		Base:    "upstream/main",
		Head:    "feature",
		Target:  Target{Handle: "owner.test", Repo: "target"},
		Source:  &source,
	})
	if err != nil {
		t.Fatalf("CreatePull() error = %v", err)
	}
	if result.Base != "main" || git.patchBase != "base-commit" || git.patchHead != "feature" {
		t.Fatalf("CreatePull() result = %+v, patch range = %q..%q", result, git.patchBase, git.patchHead)
	}
	if len(pds.puts) != 1 {
		t.Fatalf("record writes = %d, want 1", len(pds.puts))
	}
	record, ok := pds.puts[0].Record.(tangledlex.RepoPull)
	if !ok || record.Target == nil || record.Target.Branch != "main" {
		t.Fatalf("pull record = %+v, want target branch main", pds.puts[0].Record)
	}
}

func TestCreatePullRequiresExplicitBaseForFork(t *testing.T) {
	pds := &testPDS{}
	git := &testGit{}
	service := testService(pds, git, &testKnot{})
	service.appview = testAppview{repos: map[string]*tangled.Repo{
		"at://did:plc:owner/sh.tangled.repo/target": {
			URI:   "at://did:plc:owner/sh.tangled.repo/target",
			Value: tangledlex.Repo{RepoDid: optionalString("did:plc:target-repo")},
		},
		"at://did:plc:owner/sh.tangled.repo/fork": {
			URI:   "at://did:plc:owner/sh.tangled.repo/fork",
			Value: tangledlex.Repo{RepoDid: optionalString("did:plc:fork-repo")},
		},
	}}
	source := Target{Handle: "owner.test", Repo: "fork"}

	_, err := service.CreatePull(context.Background(), CreatePullInput{
		RepoDir: "/tmp/repo",
		Title:   "Example",
		Head:    "feature",
		Target:  Target{Handle: "owner.test", Repo: "target"},
		Source:  &source,
	})
	if err == nil || !strings.Contains(err.Error(), "fork pull requests require --base") {
		t.Fatalf("CreatePull() error = %v, want explicit fork base error", err)
	}
	if len(pds.puts) != 0 || git.patchBase != "" {
		t.Fatalf("CreatePull() wrote %d records and generated patch from %q", len(pds.puts), git.patchBase)
	}
}

func TestUpdatePullRoundAppendsWithCompareAndSwap(t *testing.T) {
	oldCID := cid.MustParse("bafybeigdyrzt5m6b5nkn55vsgzzfw5cfs2tidw6zqugycdkyybf2z7kz4q")
	newCID := cid.MustParse("bafybeibwzifrf5tfwmbtw6ewjqr5q6rh5y5b6gzzledmxce5ilrjzsozoa")
	oldBlob := lexutil.LexBlob{MimeType: "application/gzip", Ref: lexutil.LexLink(oldCID)}
	pds := &testPDS{
		record: &atproto.GetRecordOutput{
			CID: func() *syntax.CID { value := syntax.CID("bafyreicurrent"); return &value }(),
			Value: tangledlex.RepoPull{
				LexiconTypeID: pullCollection,
				Title:         "Example",
				CreatedAt:     "2026-07-29T00:00:00Z",
				Target:        &tangledlex.RepoPull_Target{Repo: "did:plc:target", Branch: "main"},
				Source:        &tangledlex.RepoPull_Source{Repo: optionalString("did:plc:source"), Branch: "feature"},
				Rounds:        []*tangledlex.RepoPull_Round{{CreatedAt: "2026-07-29T00:00:00Z", PatchBlob: &oldBlob}},
			},
		},
		uploadBlob: &atproto.Blob{Type: "blob", MimeType: "application/gzip", Ref: lexutil.LexLink(newCID), Size: 42},
	}
	git := &testGit{
		patch: []byte("diff --git a/a b/a\n"),
		pullBases: map[string]gitutil.PullBase{
			"upstream/main": {Revision: "base-commit", Branch: "main"},
		},
	}
	service := testService(pds, git, &testKnot{})

	if err := service.UpdatePullRound(context.Background(), UpdatePullRoundInput{
		RepoDir: "/tmp/repo", Rkey: "pr-1", Base: "upstream/main",
	}); err != nil {
		t.Fatalf("UpdatePullRound() error = %v", err)
	}
	if git.patchBase != "base-commit" || git.patchHead != "feature" {
		t.Fatalf("patch range = %q..%q, want base-commit..feature", git.patchBase, git.patchHead)
	}
	if len(pds.puts) != 1 {
		t.Fatalf("record writes = %d, want 1", len(pds.puts))
	}
	put := pds.puts[0]
	if put.SwapRecord == nil || put.SwapRecord.String() != "bafyreicurrent" {
		t.Fatalf("SwapRecord = %v, want bafyreicurrent", put.SwapRecord)
	}
	record, ok := put.Record.(tangledlex.RepoPull)
	if !ok {
		t.Fatalf("record type = %T, want RepoPull", put.Record)
	}
	if len(record.Rounds) != 2 {
		t.Fatalf("round count = %d, want 2", len(record.Rounds))
	}
	if record.Rounds[0].PatchBlob.Ref.String() != oldCID.String() || record.Rounds[1].PatchBlob.Ref.String() != newCID.String() {
		t.Fatalf("rounds = %+v", record.Rounds)
	}
}

func TestUpdatePullRoundRequiresExplicitBaseForFork(t *testing.T) {
	pds := pullRoundPDS("did:plc:target", "did:plc:source")
	git := &testGit{}
	service := testService(pds, git, &testKnot{})

	err := service.UpdatePullRound(context.Background(), UpdatePullRoundInput{RepoDir: "/tmp/repo", Rkey: "pr-1"})
	if err == nil || !strings.Contains(err.Error(), "fork pull requests require --base") {
		t.Fatalf("UpdatePullRound() error = %v, want explicit fork base error", err)
	}
	if len(pds.puts) != 0 || len(git.pullBaseInputs) != 0 {
		t.Fatalf("UpdatePullRound() wrote %d records and resolved bases %v", len(pds.puts), git.pullBaseInputs)
	}
}

func TestUpdatePullRoundDefaultsSameRepositoryToOriginTargetBranch(t *testing.T) {
	pds := pullRoundPDS("did:plc:target", "did:plc:target")
	git := &testGit{
		patch: []byte("patch"),
		pullBases: map[string]gitutil.PullBase{
			"origin/main": {Revision: "origin-main", Branch: "main"},
		},
	}
	service := testService(pds, git, &testKnot{})

	err := service.UpdatePullRound(context.Background(), UpdatePullRoundInput{RepoDir: "/tmp/repo", Rkey: "pr-1"})
	if err != nil {
		t.Fatalf("UpdatePullRound() error = %v", err)
	}
	if len(git.pullBaseInputs) != 1 || git.pullBaseInputs[0] != "origin/main" {
		t.Fatalf("resolved bases = %v, want origin/main", git.pullBaseInputs)
	}
	if git.patchBase != "origin-main" || git.patchHead != "feature" {
		t.Fatalf("patch range = %q..%q, want origin-main..feature", git.patchBase, git.patchHead)
	}
}

func TestUpdatePullRoundRejectsDifferentTargetBranch(t *testing.T) {
	pds := pullRoundPDS("did:plc:target", "did:plc:source")
	git := &testGit{pullBases: map[string]gitutil.PullBase{
		"upstream/release": {Revision: "release-commit", Branch: "release"},
	}}
	service := testService(pds, git, &testKnot{})

	err := service.UpdatePullRound(context.Background(), UpdatePullRoundInput{
		RepoDir: "/tmp/repo", Rkey: "pr-1", Base: "upstream/release",
	})
	if err == nil || !strings.Contains(err.Error(), "resolves to target branch") {
		t.Fatalf("UpdatePullRound() error = %v, want target branch mismatch", err)
	}
	if len(pds.puts) != 0 || git.patchBase != "" {
		t.Fatalf("UpdatePullRound() wrote %d records and generated patch from %q", len(pds.puts), git.patchBase)
	}
}

func pullRoundPDS(targetRepoDID, sourceRepoDID string) *testPDS {
	currentCID := syntax.CID("bafyreicurrent")
	patchCID := cid.MustParse("bafybeibwzifrf5tfwmbtw6ewjqr5q6rh5y5b6gzzledmxce5ilrjzsozoa")
	return &testPDS{
		record: &atproto.GetRecordOutput{
			CID: &currentCID,
			Value: tangledlex.RepoPull{
				LexiconTypeID: pullCollection,
				Title:         "Example",
				CreatedAt:     "2026-07-29T00:00:00Z",
				Target:        &tangledlex.RepoPull_Target{Repo: targetRepoDID, Branch: "main"},
				Source:        &tangledlex.RepoPull_Source{Repo: optionalString(sourceRepoDID), Branch: "feature"},
				Rounds:        []*tangledlex.RepoPull_Round{},
			},
		},
		uploadBlob: &atproto.Blob{Type: "blob", MimeType: "application/gzip", Ref: lexutil.LexLink(patchCID), Size: 42},
	}
}
