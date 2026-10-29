package app

import (
	"context"
	"testing"

	"github.com/alyraffauf/tg/atproto"
)

type changingDeclarationPDS struct {
	*testPDS
	reads int
}

func (p *changingDeclarationPDS) GetRecord(ctx context.Context, actor, collection, key string) (*atproto.GetRecordOutput, error) {
	p.reads++
	if p.reads > 1 {
		return &atproto.GetRecordOutput{CID: cidPointer(), Value: map[string]any{"repo": "did:plc:different"}}, nil
	}
	return p.testPDS.GetRecord(ctx, actor, collection, key)
}

func TestDeleteRepoRejectsChangedDeclarationBeforeDeletingAnything(t *testing.T) {
	service, pds, origin := migrationService(t)
	service.sessions = testSessions{pds: &changingDeclarationPDS{testPDS: pds}}
	if _, err := service.DeleteRepo(context.Background(), migrationTarget()); err == nil {
		t.Fatal("deleted a changed declaration")
	}
	if len(pds.deletes) != 0 || origin.deleteCalls != 0 || pds.serviceAuthCalls != 0 {
		t.Fatalf("deletes=%v knot=%d auth=%d", pds.deletes, origin.deleteCalls, pds.serviceAuthCalls)
	}
}
