package app

import (
	"testing"
	"time"

	"github.com/alyraffauf/tg/atproto"
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
	if record.Source.Repo != "did:plc:fork" {
		t.Fatalf("unexpected source: %+v", record.Source)
	}
}
