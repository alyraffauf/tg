package knot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMergePostsCurrentInput(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/sh.tangled.repo.merge" || request.Method != http.MethodPost {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		var input MergeInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		if input.DID != "did:plc:owner" || input.Name != "tg" || input.Repo != "did:plc:repo" || input.Branch != "master" || input.Patch != "patch" {
			t.Fatalf("merge input = %+v", input)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewWithClient(strings.TrimPrefix(server.URL, "https://"), "token", server.Client())
	if err := client.Merge(context.Background(), MergeInput{
		DID: "did:plc:owner", Name: "tg", Repo: "did:plc:repo", Branch: "master", Patch: "patch",
	}); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
}
