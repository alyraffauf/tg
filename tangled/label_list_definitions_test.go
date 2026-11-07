package tangled

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestListLabelDefinitionsFollowsCursorAndSkipsForeignRecords(t *testing.T) {
	pages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		query := r.URL.Query()
		if r.URL.Path != "/xrpc/org.tangled.label.listDefinitions" || query.Get("subject") != "did:plc:repo" {
			t.Errorf("request=%s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		if query.Get("cursor") == "" {
			fmt.Fprint(w, `{"cursor":"next","items":[
				{"uri":"at://did:plc:repo/org.tangled.label.definition/bug","cid":"cid-bug","value":{"$type":"org.tangled.label.definition","name":"bug","color":"#f00","createdAt":"2026-10-01T00:00:00Z"}},
				{"uri":"at://did:plc:other/org.tangled.label.definition/spoof","value":{"$type":"org.tangled.label.definition","name":"spoof","createdAt":"2026-10-01T00:00:00Z"}}
			]}`)
			return
		}
		if query.Get("cursor") != "next" {
			t.Errorf("cursor=%q", query.Get("cursor"))
		}
		fmt.Fprint(w, `{"items":[{"uri":"at://did:plc:repo/org.tangled.label.definition/docs","value":{"$type":"org.tangled.label.definition","name":"docs","createdAt":"2026-10-01T00:00:00Z"}}]}`)
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}

	result, err := client.ListLabelDefinitions(context.Background(), "did:plc:repo")
	if err != nil {
		t.Fatal(err)
	}
	want := []LabelDefinition{
		{URI: "at://did:plc:repo/org.tangled.label.definition/bug", CID: "cid-bug", Key: "bug", Name: "bug", Color: "#f00"},
		{URI: "at://did:plc:repo/org.tangled.label.definition/docs", Key: "docs", Name: "docs"},
	}
	if pages != 2 || !reflect.DeepEqual(result.Items, want) {
		t.Fatalf("pages=%d items=%+v", pages, result.Items)
	}
	if len(result.Warnings) != 1 || result.Warnings[0].URI != "at://did:plc:other/org.tangled.label.definition/spoof" {
		t.Fatalf("warnings=%+v", result.Warnings)
	}
}

func TestDetailedTicketKeepsAppliedLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"uri":"at://did:plc:repo/org.tangled.track.ticket/key","cid":"cid","author":{"did":"did:plc:alice"},"title":{"text":"Title"},"state":"open","tangledLabels":[{"$type":"org.tangled.label.defs#labelToken","key":"bug"}],"createdAt":"2026-10-01T00:00:00Z"}`)
	}))
	defer server.Close()
	client := Tangled{Client: &atclient.APIClient{Host: server.URL, Client: server.Client()}}
	item, err := client.GetTicket(context.Background(), "at://did:plc:repo/org.tangled.track.ticket/key")
	if err != nil {
		t.Fatal(err)
	}
	var ticket Ticket
	if err := ticket.UnmarshalJSON(item.Value); err != nil {
		t.Fatal(err)
	}
	if want := []LabelToken{{Type: LabelTokenType, Key: "bug"}}; !reflect.DeepEqual(ticket.TangledLabels, want) {
		t.Fatalf("labels=%+v", ticket.TangledLabels)
	}
}
