package spindle

import (
	"context"
	"encoding/json"
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

func TestCancelPipelineAuthenticatesAndPostsInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/sh.tangled.ci.cancelPipeline" || request.Method != http.MethodPost {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		var input CancelPipelineInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		if input.Pipeline != "3mrvk5dbnep22" || input.Repo != "did:plc:repo" || len(input.Workflows) != 1 {
			t.Fatalf("input = %+v", input)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewWithToken(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewWithToken() error = %v", err)
	}
	if err := client.CancelPipeline(context.Background(), CancelPipelineInput{
		Pipeline: "3mrvk5dbnep22", Repo: "did:plc:repo", Workflows: []string{"test.yml"},
	}); err != nil {
		t.Fatalf("CancelPipeline() error = %v", err)
	}
}

func TestTriggerPipelineAuthenticatesAndReturnsPipeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/sh.tangled.ci.triggerPipeline" || request.Method != http.MethodPost {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		var input TriggerPipelineInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		if input.Trigger.SHA != "0123456789abcdef0123456789abcdef01234567" || input.Trigger.Ref != "refs/heads/feature" {
			t.Fatalf("input = %+v", input)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"pipeline":"at://did:plc:spindle/sh.tangled.ci.pipeline/3mrvk5dbnep22"}`))
	}))
	defer server.Close()

	client, err := NewWithToken(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewWithToken() error = %v", err)
	}
	output, err := client.TriggerPipeline(context.Background(), TriggerPipelineInput{
		Repo:    "did:plc:repo",
		Trigger: ManualTrigger{LexiconTypeID: "sh.tangled.ci.trigger#manual", SHA: "0123456789abcdef0123456789abcdef01234567", Ref: "refs/heads/feature"},
	})
	if err != nil {
		t.Fatalf("TriggerPipeline() error = %v", err)
	}
	if output.Pipeline != "at://did:plc:spindle/sh.tangled.ci.pipeline/3mrvk5dbnep22" {
		t.Fatalf("output = %+v", output)
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
