package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestEditRepoGuardsRecordUpdateWithFetchedCID(t *testing.T) {
	pds := &testPDS{record: &atproto.GetRecordOutput{
		CID:   mustParseCID(t, "bafyreifetched"),
		Value: map[string]any{"$type": repoCollection, "description": "old", "knot": "knot.example", "createdAt": "2026-07-25T12:00:00Z"},
	}}
	service := editRepoTestService(pds)
	description := "new"

	_, err := service.EditRepo(context.Background(), Target{Handle: "owner.test", Repo: "example"}, EditRepoInput{Description: &description})
	if err != nil {
		t.Fatalf("EditRepo() error = %v", err)
	}
	if len(pds.puts) != 1 {
		t.Fatalf("EditRepo() writes = %d, want 1", len(pds.puts))
	}
	if pds.puts[0].SwapRecord == nil || pds.puts[0].SwapRecord.String() != "bafyreifetched" {
		t.Fatalf("EditRepo() swapRecord = %v, want bafyreifetched", pds.puts[0].SwapRecord)
	}
}

func TestEditRepoRequiresFetchedCID(t *testing.T) {
	pds := &testPDS{record: &atproto.GetRecordOutput{
		Value: map[string]any{"$type": repoCollection, "description": "old", "knot": "knot.example", "createdAt": "2026-07-25T12:00:00Z"},
	}}
	service := editRepoTestService(pds)
	description := "new"

	_, err := service.EditRepo(context.Background(), Target{Handle: "owner.test", Repo: "example"}, EditRepoInput{Description: &description})
	if err == nil || !strings.Contains(err.Error(), "omitted record CID") {
		t.Fatalf("EditRepo() error = %v, want missing CID error", err)
	}
	if len(pds.puts) != 0 {
		t.Fatalf("EditRepo() writes = %d, want 0", len(pds.puts))
	}
}

func TestEditRepoSurfacesRecordUpdateConflict(t *testing.T) {
	conflict := errors.New("InvalidSwap: record CID did not match")
	pds := &testPDS{
		record: &atproto.GetRecordOutput{CID: mustParseCID(t, "bafyreistale"), Value: map[string]any{"$type": repoCollection, "knot": "knot.example", "createdAt": "2026-07-25T12:00:00Z"}},
		putErr: conflict,
	}
	service := editRepoTestService(pds)
	description := "new"

	_, err := service.EditRepo(context.Background(), Target{Handle: "owner.test", Repo: "example"}, EditRepoInput{Description: &description})
	if err == nil || !errors.Is(err, conflict) || !strings.Contains(err.Error(), "edit repository") {
		t.Fatalf("EditRepo() error = %v, want wrapped update conflict", err)
	}
	if len(pds.puts) != 1 {
		t.Fatalf("EditRepo() writes = %d, want 1", len(pds.puts))
	}
}

func editRepoTestService(pds *testPDS) *Service {
	service := testService(pds, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{
		URI: "at://did:plc:owner/sh.tangled.repo/example",
	}}
	return service
}

func mustParseCID(t *testing.T, raw string) *syntax.CID {
	t.Helper()
	cid, err := syntax.ParseCID(raw)
	if err != nil {
		t.Fatalf("parse test CID %q: %v", raw, err)
	}
	return &cid
}
