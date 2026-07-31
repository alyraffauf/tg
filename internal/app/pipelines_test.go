package app

import (
	"context"
	"errors"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/spindle"
	"github.com/alyraffauf/tg/tangled"
)

func TestListPipelinePagesFollowsCursor(t *testing.T) {
	client := &testPipelineClient{responses: []*spindle.QueryPipelinesOutput{
		{Pipelines: []spindle.Pipeline{{ID: "first", Commit: "abc"}}, Cursor: "next"},
		{Pipelines: []spindle.Pipeline{{ID: "second", Commit: "def"}}},
	}}

	pipelines, err := listPipelinePages(context.Background(), client, "did:plc:repo")
	if err != nil {
		t.Fatalf("listPipelinePages() error = %v", err)
	}
	if len(pipelines) != 2 || pipelines[0].ID != "first" || pipelines[1].ID != "second" {
		t.Fatalf("listPipelinePages() = %+v", pipelines)
	}
	if len(client.cursors) != 2 || client.cursors[0] != "" || client.cursors[1] != "next" {
		t.Fatalf("cursors = %q, want [\"\" \"next\"]", client.cursors)
	}
}

func TestListPipelinePagesReturnsClientError(t *testing.T) {
	client := &testPipelineClient{err: errors.New("unavailable")}
	_, err := listPipelinePages(context.Background(), client, "did:plc:repo")
	if err == nil || err.Error() != "unavailable" {
		t.Fatalf("listPipelinePages() error = %v, want unavailable", err)
	}
}

func TestFindPipeline(t *testing.T) {
	pipelines := []Pipeline{{ID: "first"}, {ID: "second"}}
	found, err := findPipeline(pipelines, "second")
	if err != nil {
		t.Fatalf("findPipeline() error = %v", err)
	}
	if found.ID != "second" {
		t.Fatalf("findPipeline() = %+v, want second", found)
	}
}

func TestPipelineHasFailures(t *testing.T) {
	tests := []struct {
		name      string
		workflows []PipelineWorkflow
		want      bool
	}{
		{name: "success", workflows: []PipelineWorkflow{{Status: "success"}}, want: false},
		{name: "running", workflows: []PipelineWorkflow{{Status: "running"}}, want: false},
		{name: "failed", workflows: []PipelineWorkflow{{Status: "failed"}}, want: true},
		{name: "timeout", workflows: []PipelineWorkflow{{Status: "timeout"}}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pipelineHasFailures(Pipeline{Workflows: test.workflows}); got != test.want {
				t.Fatalf("pipelineHasFailures() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestPipelineStatusReturnsLatestPipeline(t *testing.T) {
	client := &testPipelineClient{responses: []*spindle.QueryPipelinesOutput{{
		Pipelines: []spindle.Pipeline{{
			ID: "latest", Commit: "abc", Workflows: []spindle.Workflow{{Status: "failed"}},
		}},
	}}}
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		Knot: "knot.example", Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	status, err := service.PipelineStatus(context.Background(), Target{Handle: "owner.test", Repo: "example"})
	if err != nil {
		t.Fatalf("PipelineStatus() error = %v", err)
	}
	if status.Commit != "abc" || status.Pipeline.ID != "latest" || !status.HasFailures {
		t.Fatalf("PipelineStatus() = %+v", status)
	}
}

type testPipelineClient struct {
	responses []*spindle.QueryPipelinesOutput
	cursors   []string
	err       error
}

func (c *testPipelineClient) QueryLatestPipeline(_ context.Context, _ string) (*spindle.QueryPipelinesOutput, error) {
	return c.responses[0], nil
}

type testSpindleFactory struct {
	client pipelineClient
}

func (f testSpindleFactory) New(string) (pipelineClient, error) {
	return f.client, nil
}

func (c *testPipelineClient) QueryPipelines(_ context.Context, _ string, cursor string) (*spindle.QueryPipelinesOutput, error) {
	c.cursors = append(c.cursors, cursor)
	if c.err != nil {
		return nil, c.err
	}
	response := c.responses[0]
	c.responses = c.responses[1:]
	return response, nil
}
