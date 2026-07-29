package app

import (
	"context"
	"testing"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/tangledlex"
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
	git := &testGit{patch: []byte("diff --git a/a b/a\n")}
	service := testService(pds, git, &testKnot{})

	if err := service.UpdatePullRound(context.Background(), "/tmp/repo", "pr-1"); err != nil {
		t.Fatalf("UpdatePullRound() error = %v", err)
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
