package tangled

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

func TestGetRepoByDID(t *testing.T) {
	manifest := `{"uri":"at://did:plc:repository/org.tangled.repo.manifest/self","cid":"bafyrei","value":{"$type":"org.tangled.repo.manifest","declaration":{"name":"example","owner":"did:plc:owner"},"knot":"knot.example","createdAt":"2026-10-31T00:00:00Z"}}`
	legacy := `{"uri":"at://did:plc:owner/sh.tangled.repo/example","cid":"bafyrei","value":{"$type":"sh.tangled.repo","name":"example","knot":"knot.example","repoDid":"did:plc:repository","createdAt":"2026-08-27T00:00:00Z"}}`

	tests := []struct {
		name    string
		handler func(t *testing.T, writer http.ResponseWriter, request *http.Request)
	}{
		{
			name: "org.tangled appview",
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/xrpc/org.tangled.repo.getRepoByRepoDid" {
					t.Fatalf("request path = %q", request.URL.Path)
				}
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(manifest))
			},
		},
		{
			name: "legacy appview",
			handler: func(t *testing.T, writer http.ResponseWriter, request *http.Request) {
				switch request.URL.Path {
				case "/xrpc/org.tangled.repo.getRepoByRepoDid":
					writer.WriteHeader(http.StatusNotImplemented)
					_, _ = writer.Write([]byte(`{"error":"MethodNotImplemented","message":"Method Not Implemented"}`))
				case "/xrpc/sh.tangled.repo.getRepoByRepoDid":
					writer.Header().Set("Content-Type", "application/json")
					_, _ = writer.Write([]byte(legacy))
				default:
					t.Fatalf("request path = %q", request.URL.Path)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if got := request.URL.Query().Get("repoDid"); got != "did:plc:repository" {
					t.Fatalf("repoDid = %q", got)
				}
				test.handler(t, writer, request)
			}))
			defer server.Close()

			client := Tangled{Client: &atclient.APIClient{Client: server.Client(), Host: server.URL}}
			repo, err := client.GetRepoByDID(context.Background(), "did:plc:repository")
			if err != nil {
				t.Fatalf("GetRepoByDID() error = %v", err)
			}
			if repo.Value.Name == nil || *repo.Value.Name != "example" {
				t.Fatalf("repo name = %v, want example", repo.Value.Name)
			}
			if repo.Value.Knot != "knot.example" {
				t.Fatalf("repo knot = %q, want knot.example", repo.Value.Knot)
			}
			if repo.Value.RepoDid == nil || *repo.Value.RepoDid != "did:plc:repository" {
				t.Fatalf("repo DID = %v, want did:plc:repository", repo.Value.RepoDid)
			}
		})
	}
}
