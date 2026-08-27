package knot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDescribeRepo(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/xrpc/sh.tangled.repo.describeRepo" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if got := request.URL.Query().Get("repoDid"); got != "did:plc:repository" {
			t.Fatalf("repoDid = %q", got)
		}
		_, _ = writer.Write([]byte(`{"repoDid":"did:plc:repository","ownerDid":"did:plc:owner","rkey":"repo"}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := NewPublicWithClient(serverURL.Host, server.Client())
	description, err := client.DescribeRepo(context.Background(), "did:plc:repository")
	if err != nil {
		t.Fatalf("DescribeRepo() error = %v", err)
	}
	if description.RepoDID != "did:plc:repository" || description.OwnerDID != "did:plc:owner" || description.RKey != "repo" {
		t.Fatalf("DescribeRepo() = %+v", description)
	}
}

func TestDescribeRepoWrapsKnotError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, `{"error":"RepoNotFound"}`, http.StatusNotFound)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := NewPublicWithClient(serverURL.Host, server.Client())
	_, err = client.DescribeRepo(context.Background(), "did:plc:missing")
	if err == nil || !strings.Contains(err.Error(), `describe repository "did:plc:missing"`) {
		t.Fatalf("DescribeRepo() error = %v", err)
	}
}
