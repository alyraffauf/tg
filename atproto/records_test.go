package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		Collection: "org.tangled.repo.declaration",
		Record: tangledlex.RepoDeclaration{
			LexiconTypeID: "org.tangled.repo.declaration",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "repo") {
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
		if !ok || record["$type"] != "org.tangled.repo.declaration" {
			t.Fatalf("record = %#v, want serialized Tangled string", input.Record)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"uri":"at://did:plc:abc123/org.tangled.repo.declaration/3k2abc"}`))
	}))
	defer server.Close()

	client := &ATProto{Client: &atclient.APIClient{Client: server.Client(), Host: server.URL}}
	uri, _, err := client.PutRecord(context.Background(), PutRecordInput{
		Repo:       "did:plc:abc123",
		Collection: "org.tangled.repo.declaration",
		Rkey:       "3k2abc",
		Record: tangledlex.RepoDeclaration{
			LexiconTypeID: "org.tangled.repo.declaration",
			Repo:          "did:plc:repo",
			CreatedAt:     "2026-07-25T12:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("PutRecord() error = %v", err)
	}
	if uri != "at://did:plc:abc123/org.tangled.repo.declaration/3k2abc" {
		t.Fatalf("PutRecord() URI = %q", uri)
	}
}

func TestRecordUpdateRoundTripsFetchedCIDAsSwapRecord(t *testing.T) {
	var swapRecord string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.Method {
		case http.MethodGet:
			_, _ = writer.Write([]byte(`{"uri":"at://did:plc:abc123/org.tangled.repo.declaration/example","cid":"bafyreifetched","value":{"$type":"org.tangled.repo.declaration","repo":"did:plc:repo","createdAt":"2026-07-25T12:00:00Z"}}`))
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
			_, _ = writer.Write([]byte(`{"uri":"at://did:plc:abc123/org.tangled.repo.declaration/example","cid":"bafyreinew"}`))
		default:
			t.Errorf("request method = %s, want GET or POST", request.Method)
		}
	}))
	defer server.Close()

	client := &ATProto{Client: &atclient.APIClient{Client: server.Client(), Host: server.URL}}
	found, err := client.GetRecord(context.Background(), "did:plc:abc123", "org.tangled.repo.declaration", "example")
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if found.CID == nil || found.CID.String() != "bafyreifetched" {
		t.Fatalf("GetRecord() CID = %v, want bafyreifetched", found.CID)
	}

	_, _, err = client.PutRecord(context.Background(), PutRecordInput{
		Repo:       "did:plc:abc123",
		Collection: "org.tangled.repo.declaration",
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

type diagnosticBody struct {
	read   int
	closed bool
}

func (body *diagnosticBody) Read(data []byte) (int, error) {
	if body.read >= 1<<20 {
		return 0, io.EOF
	}
	count := min(len(data), (1<<20)-body.read)
	for i := range data[:count] {
		data[i] = 'x'
	}
	body.read += count
	return count, nil
}
func (body *diagnosticBody) Close() error { body.closed = true; return nil }

func TestUploadBlobBoundsErrorDiagnosticsAndClosesBody(t *testing.T) {
	body := &diagnosticBody{}
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: body, Header: make(http.Header)}, nil
	})
	client := &ATProto{Client: &atclient.APIClient{Client: &http.Client{Transport: transport}, Host: "https://pds.example"}}
	_, err := client.UploadBlob(context.Background(), []byte("content"), "text/plain")
	if err == nil || !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("missing bounded HTTP diagnostic; error length=%d", len(fmt.Sprint(err)))
	}
	if body.read > (64<<10)+1 || len(err.Error()) > (64<<10)+200 || !body.closed {
		t.Fatalf("read=%d error bytes=%d closed=%t", body.read, len(err.Error()), body.closed)
	}
}
