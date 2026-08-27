package tangled

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestGetRepoByDID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/sh.tangled.repo.getRepoByRepoDid" {
			t.Fatalf("request path = %q", request.URL.Path)
		}
		if got := request.URL.Query().Get("repoDid"); got != "did:plc:repository" {
			t.Fatalf("repoDid = %q", got)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"uri":"at://did:plc:owner/sh.tangled.repo/example","value":{"$type":"sh.tangled.repo","knot":"knot.example","repoDid":"did:plc:repository","createdAt":"2026-08-27T00:00:00Z"}}`))
	}))
	defer server.Close()

	client := Tangled{Client: &atclient.APIClient{Client: server.Client(), Host: server.URL}}
	repo, err := client.GetRepoByDID(context.Background(), "did:plc:repository")
	if err != nil {
		t.Fatalf("GetRepoByDID() error = %v", err)
	}
	if repo.URI != "at://did:plc:owner/sh.tangled.repo/example" || repo.Value.RepoDid == nil || *repo.Value.RepoDid != "did:plc:repository" {
		t.Fatalf("GetRepoByDID() = %+v", repo)
	}
}
