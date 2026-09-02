package tangled

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestListIssuesRejectsUnexpectedRecordType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/sh.tangled.repo.listIssues" {
			t.Fatalf("unexpected request path %q", request.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"uri":"at://did:plc:repo/sh.tangled.repo.issue/abc","value":{"$type":"sh.tangled.repo.pull"},"state":"open","commentCount":0}]}`))
	}))
	defer server.Close()

	client := Tangled{Client: &atclient.APIClient{Client: server.Client(), Host: server.URL}}
	result, err := client.ListIssues(context.Background(), "did:plc:repo", ListOpts{})
	if err != nil {
		t.Fatalf("ListIssues error = %v", err)
	}
	if len(result.Items) != 0 || len(result.Warnings) != 1 {
		t.Fatalf("ListIssues() = %+v, want one warning and no items", result)
	}
}
