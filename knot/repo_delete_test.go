package knot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeleteRepoPostsRepositoryDID(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/xrpc/sh.tangled.repo.delete" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if body["repo"] != "did:plc:repository" {
			t.Errorf("repo = %v, want repository DID", body["repo"])
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWithClient(strings.TrimPrefix(server.URL, "https://"), "token", server.Client())
	err := client.DeleteRepo(context.Background(), DeleteRepoInput{
		Repo: "did:plc:repository", DID: "did:plc:owner", Name: "example", Rkey: "record-key",
	})
	if err != nil {
		t.Fatalf("DeleteRepo() error = %v", err)
	}
}
