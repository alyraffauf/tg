package app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/tangled"
)

func TestLegacyTicketTitleSurvivesFosterReadsAndStateEdits(t *testing.T) {
	service, _, origin := migrationService(t)
	raw := json.RawMessage(`{"$type":"org.tangled.track.ticket","title":"Historical title","state":"open","createdAt":"2026-10-01T00:00:00Z"}`)
	uri := "at://did:plc:repo/" + tangled.IssueCollection + "/ticket"
	view := service.appview.(testAppview)
	view.ticket = &tangled.ListItem{URI: uri, Value: raw, Author: "did:plc:alice"}
	service.appview = view
	service.fosterKnot = "foster.example"
	origin.legacy = true
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	origin.records = map[string]*atproto.GetRecordOutput{tangled.IssueCollection + "/ticket": {URI: uri, CID: cidPointer(), Value: value}}
	result, err := service.ViewIssue(context.Background(), migrationTarget(), "ticket")
	if err != nil || result.Title != "Historical title" || result.Author.DID != "did:plc:alice" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	origin.legacy = false
	if _, err := service.SetIssueState(context.Background(), migrationTarget(), "ticket", "closed"); err != nil {
		t.Fatal(err)
	}
	saved := origin.procedureInputs[0]["record"].(map[string]any)
	if saved["title"].(map[string]any)["text"] != "Historical title" || saved["state"] != "closed" {
		t.Fatalf("record=%+v", saved)
	}
}
