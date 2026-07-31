package app

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
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

func TestViewPipelineFetchesPipelineDirectly(t *testing.T) {
	client := &testPipelineClient{pipeline: &spindle.Pipeline{ID: "second", Repo: "did:plc:repo"}}
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	found, err := service.ViewPipeline(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "second")
	if err != nil {
		t.Fatalf("ViewPipeline() error = %v", err)
	}
	if found.ID != "second" {
		t.Fatalf("ViewPipeline() = %+v, want second", found)
	}
	if client.pipelineID != "second" {
		t.Fatalf("GetPipeline() ID = %q, want second", client.pipelineID)
	}
}

func TestViewPipelineRejectsPipelineFromAnotherRepository(t *testing.T) {
	client := &testPipelineClient{pipeline: &spindle.Pipeline{ID: "second", Repo: "did:plc:other"}}
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	_, err := service.ViewPipeline(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "second")
	if err == nil || err.Error() != "pipeline \"second\" does not belong to repository \"owner.test/example\"" {
		t.Fatalf("ViewPipeline() error = %v", err)
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

func TestSelectCancellableWorkflows(t *testing.T) {
	workflows := []spindle.Workflow{
		{Name: "pending.yml", Status: "pending"},
		{Name: "running.yml", Status: "running"},
		{Name: "done.yml", Status: "success"},
	}
	if got := selectCancellableWorkflows(workflows, nil); !slices.Equal(got, []string{"pending.yml", "running.yml"}) {
		t.Fatalf("all cancellable workflows = %q", got)
	}
	if got := selectCancellableWorkflows(workflows, []string{"running.yml", "done.yml"}); !slices.Equal(got, []string{"running.yml"}) {
		t.Fatalf("selected cancellable workflows = %q", got)
	}
}

func TestPipelineStatusReturnsLatestPipeline(t *testing.T) {
	client := &testPipelineClient{responses: []*spindle.QueryPipelinesOutput{{
		Pipelines: []spindle.Pipeline{{
			ID: "latest", Commit: "abc", Workflows: []spindle.Workflow{{Status: "failed"}},
		}},
	}}}
	service := testService(&testPDS{}, &testGit{}, &testKnot{defaultBranch: &knot.DefaultBranch{Name: "main", Hash: "abc"}})
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

func TestCancelPipelineMintsSpindleToken(t *testing.T) {
	client := &testPipelineClient{pipeline: &spindle.Pipeline{Workflows: []spindle.Workflow{
		{Name: "test.yml", Status: "pending"},
		{Name: "done.yml", Status: "success"},
	}}}
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		Knot: "knot.example", Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	result, err := service.CancelPipeline(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "3mrvk5dbnep22", []string{"test.yml", "done.yml", "unknown.yml"})
	if err != nil {
		t.Fatalf("CancelPipeline() error = %v", err)
	}
	if result.Pipeline != "3mrvk5dbnep22" || len(result.Workflows) != 1 {
		t.Fatalf("CancelPipeline() = %+v", result)
	}
	if pds.serviceAuthAudiences[0] != "did:web:spindle.example" || pds.serviceAuthLexiconMethods[0] != "sh.tangled.ci.cancelPipeline" {
		t.Fatalf("service auth = audience %q, method %q", pds.serviceAuthAudiences[0], pds.serviceAuthLexiconMethods[0])
	}
	if client.cancelInput.Pipeline != "3mrvk5dbnep22" || client.cancelInput.Repo != "did:plc:repo" || !slices.Equal(client.cancelInput.Workflows, []string{"test.yml"}) {
		t.Fatalf("cancel input = %+v", client.cancelInput)
	}
}

func TestTriggerPipelineUsesFullSHAWithoutGitResolution(t *testing.T) {
	commit := "0123456789abcdef0123456789abcdef01234567"
	client := &testPipelineClient{triggerOutput: &spindle.TriggerPipelineOutput{Pipeline: "at://did:plc:spindle/sh.tangled.ci.pipeline/3mrvk5dbnep22"}}
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangledlex.Repo{
		Knot: "knot.example", Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	result, err := service.TriggerPipeline(context.Background(), Target{Handle: "owner.test", Repo: "example"}, commit, []string{"test.yml"})
	if err != nil {
		t.Fatalf("TriggerPipeline() error = %v", err)
	}
	if result.Pipeline != "3mrvk5dbnep22" || result.Commit != commit {
		t.Fatalf("TriggerPipeline() = %+v", result)
	}
	if pds.serviceAuthLexiconMethods[0] != "sh.tangled.ci.triggerPipeline" {
		t.Fatalf("service auth method = %q", pds.serviceAuthLexiconMethods[0])
	}
	if client.triggerInput.Trigger.SHA != commit || client.triggerInput.Trigger.Ref != "" || client.triggerInput.Repo != "did:plc:repo" {
		t.Fatalf("trigger input = %+v", client.triggerInput)
	}
}

type testPipelineClient struct {
	responses     []*spindle.QueryPipelinesOutput
	cursors       []string
	err           error
	cancelInput   spindle.CancelPipelineInput
	pipeline      *spindle.Pipeline
	pipelineID    string
	triggerInput  spindle.TriggerPipelineInput
	triggerOutput *spindle.TriggerPipelineOutput
	logEvents     []spindle.PipelineLogEvent
}

func (c *testPipelineClient) QueryLatestPipeline(_ context.Context, _ string) (*spindle.QueryPipelinesOutput, error) {
	return c.responses[0], nil
}

func (c *testPipelineClient) GetPipeline(_ context.Context, pipelineID string) (*spindle.Pipeline, error) {
	c.pipelineID = pipelineID
	return c.pipeline, c.err
}

func (c *testPipelineClient) CancelPipeline(_ context.Context, input spindle.CancelPipelineInput) error {
	c.cancelInput = input
	return c.err
}

func (c *testPipelineClient) TriggerPipeline(_ context.Context, input spindle.TriggerPipelineInput) (*spindle.TriggerPipelineOutput, error) {
	c.triggerInput = input
	return c.triggerOutput, c.err
}

type testSpindleFactory struct {
	client pipelineClient
}

func (f testSpindleFactory) New(string) (pipelineClient, error) {
	return f.client, nil
}

func (f testSpindleFactory) NewWithToken(string, string) (pipelineClient, error) {
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

func (c *testPipelineClient) SubscribePipelineLogs(_ context.Context, _ string, _ []string, onEvent func(spindle.PipelineLogEvent) error) error {
	for _, event := range c.logEvents {
		if err := onEvent(event); err != nil {
			return err
		}
	}
	return nil
}
