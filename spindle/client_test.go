package spindle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestQueryPipelinesUsesSpindleEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/org.tangled.ci.queryPipelines" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("repo") != "did:plc:repo" || request.URL.Query().Get("limit") != "250" {
			t.Fatalf("query = %q", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"pipelines":[{"id":"pipeline-1","commit":"abc","trigger":{"$type":"org.tangled.ci.trigger#push"},"workflows":[{"id":"test","name":"test","status":"success"}]}],"total":1}`))
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
		if request.URL.Path != "/xrpc/org.tangled.ci.cancelPipeline" || request.Method != http.MethodPost {
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
		if request.URL.Path != "/xrpc/org.tangled.event.notifyEvent" || request.Method != http.MethodPost {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		var input map[string]any
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Fatalf("decode input: %v", err)
		}
		event, ok := input["event"].(map[string]any)
		if !ok || len(input) != 2 || input["repo"] != "did:plc:repo" {
			t.Fatalf("notifyEvent body = %+v", input)
		}
		if event["$type"] != "org.tangled.ci.trigger.manual" || event["repo"] != input["repo"] || event["commit"] != "0123456789abcdef0123456789abcdef01234567" || event["ref"] != "refs/heads/feature" {
			t.Fatalf("manual event = %+v", event)
		}
		workflows, ok := event["workflows"].([]any)
		if !ok || len(workflows) != 1 || workflows[0] != "test.yml" {
			t.Fatalf("workflow selection = %+v", event["workflows"])
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ci":{"pipelines":["3mrvk5dbnep22","3mrvk5dbnep23"],"message":"Started verification"}}`))
	}))
	defer server.Close()

	client, err := NewWithToken(server.URL, "token", server.Client())
	if err != nil {
		t.Fatalf("NewWithToken() error = %v", err)
	}
	output, err := client.TriggerPipeline(context.Background(), TriggerPipelineInput{
		Repo:  "did:plc:repo",
		Event: ManualTrigger{LexiconTypeID: "org.tangled.ci.trigger.manual", Repo: "did:plc:repo", SHA: "0123456789abcdef0123456789abcdef01234567", Ref: "refs/heads/feature", Workflows: []string{"test.yml"}},
	})
	if err != nil {
		t.Fatalf("TriggerPipeline() error = %v", err)
	}
	if !slices.Equal(output.Pipelines, []string{"3mrvk5dbnep22", "3mrvk5dbnep23"}) || output.Message != "Started verification" {
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

func TestGetPipelineQueriesAndDecodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/"+nsidGetPipeline.String() {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("pipeline") != "3mrvk5dbnep22" {
			t.Fatalf("pipeline query = %q", request.URL.Query().Get("pipeline"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"3mrvk5dbnep22","commit":"abc","trigger":{"$type":"org.tangled.ci.trigger#push"},"workflows":[{"id":"test","name":"test","status":"success"}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	pipeline, err := client.GetPipeline(context.Background(), "3mrvk5dbnep22")
	if err != nil {
		t.Fatalf("GetPipeline() error = %v", err)
	}
	if pipeline.ID != "3mrvk5dbnep22" || pipeline.Commit != "abc" || len(pipeline.Workflows) != 1 {
		t.Fatalf("pipeline = %+v", pipeline)
	}
}

func TestServiceDID(t *testing.T) {
	if got, err := ServiceDID("spindle.example"); err != nil || got != "did:web:spindle.example" {
		t.Fatalf("ServiceDID() = %q, %v", got, err)
	}
	if got, err := ServiceDID("https://spindle.example:443"); err != nil || got != "did:web:spindle.example%3A443" {
		t.Fatalf("ServiceDID(port) = %q, %v", got, err)
	}
	if _, err := ServiceDID(""); err == nil {
		t.Fatal("ServiceDID(empty) error = nil")
	}
}

func TestPipelineApprovalAndRetryContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("request=%s %s", r.Method, r.URL)
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input["repo"] != "did:plc:repo" || input["pipeline"] != "opaque-pipeline" {
			t.Errorf("body=%+v", input)
		}
		switch r.URL.Path {
		case "/xrpc/org.tangled.ci.approvePipeline":
			if len(input) != 2 {
				t.Errorf("approval=%+v", input)
			}
		case "/xrpc/org.tangled.ci.retryPipeline":
			workflows, ok := input["workflows"].([]any)
			if !ok || len(workflows) != 1 || workflows[0] != "build.yml" {
				t.Errorf("retry=%+v", input)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"pipeline":"new-pipeline"}`)
		default:
			t.Errorf("unexpected route=%s", r.URL)
		}
	}))
	defer server.Close()
	client, err := NewWithToken(server.URL, "token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ApprovePipeline(context.Background(), ApprovePipelineInput{Repo: "did:plc:repo", Pipeline: "opaque-pipeline"}); err != nil {
		t.Fatal(err)
	}
	retry, err := client.RetryPipeline(context.Background(), RetryPipelineInput{Repo: "did:plc:repo", Pipeline: "opaque-pipeline", Workflows: []string{"build.yml"}})
	if err != nil || retry.Pipeline != "new-pipeline" {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
}

func TestPipelineMutationsPreserveDenialsAndRejectMissingReceipts(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"denied", 403, `{"error":"AccessControl","message":"permission denied"}`},
		{"not retryable", 400, `{"error":"PipelineNotRetryable"}`},
		{"missing receipt", 200, `{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client, _ := NewWithToken(server.URL, "token", server.Client())
			if _, err := client.RetryPipeline(context.Background(), RetryPipelineInput{Repo: "did:plc:repo", Pipeline: "pipeline"}); err == nil {
				t.Fatal("retry accepted failed request")
			}
			if _, err := client.TriggerPipeline(context.Background(), TriggerPipelineInput{Repo: "did:plc:repo"}); test.status != http.StatusOK && err == nil {
				t.Fatal("trigger accepted failed request")
			}
		})
	}
}

func TestTriggerPipelineNotificationResponses(t *testing.T) {
	for _, test := range []struct {
		name      string
		body      string
		pipelines []string
		message   string
	}{
		{name: "no CI", body: `{}`},
		{name: "empty CI", body: `{"ci":{}}`},
		{name: "message only", body: `{"ci":{"message":"No matching workflows"}}`, message: "No matching workflows"},
		{name: "legacy", body: `{"pipeline":"old-pipeline","message":"Old spindle"}`, pipelines: []string{"old-pipeline"}, message: "Old spindle"},
		{name: "CI takes precedence", body: `{"ci":{"pipelines":[]},"pipeline":"obsolete"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			client, err := NewWithToken(server.URL, "token", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			result, err := client.TriggerPipeline(context.Background(), TriggerPipelineInput{Repo: "did:plc:repo"})
			if err != nil || !slices.Equal(result.Pipelines, test.pipelines) || result.Message != test.message {
				t.Fatalf("notification=%+v err=%v", result, err)
			}
		})
	}
}
