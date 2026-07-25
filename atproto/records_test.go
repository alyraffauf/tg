package atproto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestPutRecordRejectsInvalidTangledRecordBeforeRequest(t *testing.T) {
	client := &ATProto{}
	_, _, err := client.PutRecord(context.Background(), PutRecordInput{
		Collection: "sh.tangled.string",
		Record: tangledlex.String{
			LexiconTypeID: "sh.tangled.string",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "filename") {
		t.Fatalf("PutRecord() error = %v, want filename validation error", err)
	}
}

func TestPutRecordSendsValidatedTangledRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		var input PutRecordInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		record, ok := input.Record.(map[string]any)
		if !ok || record["$type"] != "sh.tangled.string" {
			t.Fatalf("record = %#v, want serialized Tangled string", input.Record)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"uri":"at://did:plc:abc123/sh.tangled.string/3k2abc"}`))
	}))
	defer server.Close()

	client := &ATProto{Client: &atclient.APIClient{Client: server.Client(), Host: server.URL}}
	uri, _, err := client.PutRecord(context.Background(), PutRecordInput{
		Repo:       "did:plc:abc123",
		Collection: "sh.tangled.string",
		Rkey:       "3k2abc",
		Record: tangledlex.String{
			LexiconTypeID: "sh.tangled.string",
			Filename:      "note.md",
			Description:   "note",
			Contents:      "text",
			CreatedAt:     "2026-07-25T12:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("PutRecord() error = %v", err)
	}
	if uri != "at://did:plc:abc123/sh.tangled.string/3k2abc" {
		t.Fatalf("PutRecord() URI = %q", uri)
	}
}

func TestRecordUpdateRoundTripsFetchedCIDAsSwapRecord(t *testing.T) {
	var swapRecord string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			_, _ = writer.Write([]byte(`{"uri":"at://did:plc:abc123/sh.tangled.repo/example","cid":"bafyreifetched","value":{"$type":"sh.tangled.repo","knot":"knot.example","createdAt":"2026-07-25T12:00:00Z"}}`))
		case http.MethodPost:
			defer request.Body.Close()
			var input struct {
				SwapRecord string `json:"swapRecord"`
			}
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
				t.Errorf("decode PutRecord request: %v", err)
				return
			}
			swapRecord = input.SwapRecord
			_, _ = writer.Write([]byte(`{"uri":"at://did:plc:abc123/sh.tangled.repo/example","cid":"bafyreinew"}`))
		default:
			t.Errorf("request method = %s, want GET or POST", request.Method)
		}
	}))
	defer server.Close()

	client := &ATProto{Client: &atclient.APIClient{Client: server.Client(), Host: server.URL}}
	found, err := client.GetRecord(context.Background(), "did:plc:abc123", "sh.tangled.repo", "example")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if found.CID == nil || found.CID.String() != "bafyreifetched" {
		t.Fatalf("GetRecord() CID = %v, want bafyreifetched", found.CID)
	}

	_, _, err = client.PutRecord(context.Background(), PutRecordInput{
		Repo:       "did:plc:abc123",
		Collection: "sh.tangled.repo",
		Rkey:       "example",
		Record:     found.Value,
		SwapRecord: found.CID,
	})
	if err != nil {
		t.Fatalf("PutRecord() error = %v", err)
	}
	if swapRecord != "bafyreifetched" {
		t.Fatalf("PutRecord() swapRecord = %q, want bafyreifetched", swapRecord)
	}
}
