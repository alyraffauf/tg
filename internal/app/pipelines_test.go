package app

import (
	"context"
	"errors"
	"testing"

	"github.com/alyraffauf/tg/spindle"
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

type testPipelineClient struct {
	responses []*spindle.QueryPipelinesOutput
	cursors   []string
	err       error
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
