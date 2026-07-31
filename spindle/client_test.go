package spindle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryPipelinesUsesSpindleEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/sh.tangled.ci.queryPipelines" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("repo") != "did:plc:repo" || request.URL.Query().Get("limit") != "250" {
			t.Fatalf("query = %q", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"pipelines":[{"id":"pipeline-1","commit":"abc","trigger":{"$type":"sh.tangled.ci.trigger#push"},"workflows":[{"id":"test","name":"test","status":"success"}]}],"total":1}`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	response, err := client.QueryPipelines(context.Background(), "did:plc:repo", "")
	if err != nil {
		t.Fatalf("QueryPipelines() error = %v", err)
	}
	if len(response.Pipelines) != 1 || response.Pipelines[0].Workflows[0].Status != "success" {
		t.Fatalf("QueryPipelines() = %+v", response)
	}
}

func TestQueryLatestPipelineLimitsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("repo") != "did:plc:repo" {
			t.Fatalf("query = %q", request.URL.RawQuery)
		}
		if request.URL.Query().Get("limit") != "1" {
			t.Fatalf("limit = %q, want 1", request.URL.Query().Get("limit"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"pipelines":[],"total":0}`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.QueryLatestPipeline(context.Background(), "did:plc:repo"); err != nil {
		t.Fatalf("QueryLatestPipeline() error = %v", err)
	}
}
