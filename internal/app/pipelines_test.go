package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

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
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{
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
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{
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
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{
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

func TestPipelineStatusFindsCommitOnLaterPage(t *testing.T) {
	client := &testPipelineClient{responses: []*spindle.QueryPipelinesOutput{
		{Pipelines: []spindle.Pipeline{{ID: "older", Commit: "old"}}, Cursor: "next"},
		{Pipelines: []spindle.Pipeline{{ID: "latest", Commit: "abc"}}},
	}}
	service := testService(&testPDS{}, &testGit{}, &testKnot{defaultBranch: &knot.DefaultBranch{Name: "main", Hash: "abc"}})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{
		Knot: "knot.example", Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	status, err := service.PipelineStatus(context.Background(), Target{Handle: "owner.test", Repo: "example"})
	if err != nil {
		t.Fatalf("PipelineStatus() error = %v", err)
	}
	if status.Pipeline.ID != "latest" || !slices.Equal(client.cursors, []string{"", "next"}) {
		t.Fatalf("status=%+v cursors=%q", status, client.cursors)
	}
}

func TestListPipelinePagesRejectsRepeatedCursor(t *testing.T) {
	client := &testPipelineClient{responses: []*spindle.QueryPipelinesOutput{{Cursor: "same"}, {Cursor: "same"}}}
	_, err := listPipelinePages(context.Background(), client, "did:plc:repo")
	if err == nil || err.Error() != "pipeline pagination repeated cursor \"same\"" {
		t.Fatalf("listPipelinePages() error = %v", err)
	}
}

func TestCancelPipelineRejectsExplicitFinishedWorkflowBeforeAuth(t *testing.T) {
	client := &testPipelineClient{pipeline: &spindle.Pipeline{Repo: "did:plc:repo", Workflows: []spindle.Workflow{{Name: "done.yml", Status: "success"}}}}
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo")}}}
	service.spindle = testSpindleFactory{client: client}

	_, err := service.CancelPipeline(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "pipeline", []string{"done.yml"})
	if err == nil || pds.serviceAuthCalls != 0 || client.cancelCalls != 0 {
		t.Fatalf("CancelPipeline() error=%v auth=%d cancel=%d", err, pds.serviceAuthCalls, client.cancelCalls)
	}
}

func TestPipelineLogsRejectsForeignPipelineBeforeSubscription(t *testing.T) {
	client := &testPipelineClient{pipeline: &spindle.Pipeline{Repo: "did:plc:foreign"}}
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo")}}}
	service.spindle = testSpindleFactory{client: client}

	err := service.PipelineLogs(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "pipeline", nil, func(PipelineLogEvent) error { return nil })
	if err == nil || client.subscribeCalls != 0 {
		t.Fatalf("PipelineLogs() error=%v subscribe=%d", err, client.subscribeCalls)
	}
}

func TestCancelPipelineMintsSpindleToken(t *testing.T) {
	client := &testPipelineClient{pipeline: &spindle.Pipeline{Repo: "did:plc:repo", Workflows: []spindle.Workflow{
		{Name: "test.yml", Status: "pending"},
		{Name: "done.yml", Status: "success"},
	}}}
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{
		Knot: "knot.example", Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	result, err := service.CancelPipeline(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "3mrvk5dbnep22", []string{"test.yml"})
	if err != nil {
		t.Fatalf("CancelPipeline() error = %v", err)
	}
	if result.Pipeline != "3mrvk5dbnep22" || len(result.Workflows) != 1 {
		t.Fatalf("CancelPipeline() = %+v", result)
	}
	if pds.serviceAuthAudiences[0] != "did:web:spindle.example" || pds.serviceAuthLexiconMethods[0] != "org.tangled.ci.cancelPipeline" {
		t.Fatalf("service auth = audience %q, method %q", pds.serviceAuthAudiences[0], pds.serviceAuthLexiconMethods[0])
	}
	if client.cancelInput.Pipeline != "3mrvk5dbnep22" || client.cancelInput.Repo != "did:plc:repo" || !slices.Equal(client.cancelInput.Workflows, []string{"test.yml"}) {
		t.Fatalf("cancel input = %+v", client.cancelInput)
	}
}

func TestTriggerPipelineUsesFullSHAWithoutGitResolution(t *testing.T) {
	commit := "0123456789abcdef0123456789abcdef01234567"
	client := &testPipelineClient{triggerOutput: &spindle.TriggerPipelineOutput{Pipelines: []string{"3mrvk5dbnep22", "3mrvk5dbnep23"}, Message: "Started verification"}}
	pds := &testPDS{}
	service := testService(pds, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{
		Knot: "knot.example", Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo"),
	}}}
	service.spindle = testSpindleFactory{client: client}

	result, err := service.TriggerPipeline(context.Background(), Target{Handle: "owner.test", Repo: "example"}, commit, []string{"test.yml"})
	if err != nil {
		t.Fatalf("TriggerPipeline() error = %v", err)
	}
	if result.Pipeline != "3mrvk5dbnep22" || !slices.Equal(result.Pipelines, client.triggerOutput.Pipelines) || result.Message != client.triggerOutput.Message || result.Commit != commit {
		t.Fatalf("TriggerPipeline() = %+v", result)
	}
	if pds.serviceAuthLexiconMethods[0] != "org.tangled.event.notifyEvent" {
		t.Fatalf("service auth method = %q", pds.serviceAuthLexiconMethods[0])
	}
	if client.triggerInput.Event.SHA != commit || client.triggerInput.Event.Ref != "" || !slices.Equal(client.triggerInput.Event.Workflows, []string{"test.yml"}) || client.triggerInput.Repo != "did:plc:repo" {
		t.Fatalf("trigger input = %+v", client.triggerInput)
	}
}

type testPipelineClient struct {
	responses      []*spindle.QueryPipelinesOutput
	cursors        []string
	err            error
	cancelInput    spindle.CancelPipelineInput
	pipeline       *spindle.Pipeline
	pipelineID     string
	triggerInput   spindle.TriggerPipelineInput
	triggerOutput  *spindle.TriggerPipelineOutput
	approveInput   spindle.ApprovePipelineInput
	retryInput     spindle.RetryPipelineInput
	retryOutput    *spindle.RetryPipelineOutput
	logEvents      []spindle.PipelineLogEvent
	cancelCalls    int
	subscribeCalls int
}

func (c *testPipelineClient) QueryLatestPipeline(_ context.Context, _ string) (*spindle.QueryPipelinesOutput, error) {
	return c.responses[0], nil
}

func (c *testPipelineClient) GetPipeline(_ context.Context, pipelineID string, _ ...string) (*spindle.Pipeline, error) {
	c.pipelineID = pipelineID
	return c.pipeline, c.err
}

func (c *testPipelineClient) CancelPipeline(_ context.Context, input spindle.CancelPipelineInput) error {
	c.cancelCalls++
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

func (c *testPipelineClient) SubscribePipelineLogs(_ context.Context, _ string, _ []string, onEvent func(spindle.PipelineLogEvent) error, _ ...string) error {
	c.subscribeCalls++
	for _, event := range c.logEvents {
		if err := onEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (c *testPipelineClient) ApprovePipeline(_ context.Context, input spindle.ApprovePipelineInput) error {
	c.approveInput = input
	return c.err
}
func (c *testPipelineClient) RetryPipeline(_ context.Context, input spindle.RetryPipelineInput) (*spindle.RetryPipelineOutput, error) {
	c.retryInput = input
	return c.retryOutput, c.err
}

func TestPipelineApprovalAndRetryUseRepositoryAuth(t *testing.T) {
	pds := &testPDS{}
	client := &testPipelineClient{retryOutput: &spindle.RetryPipelineOutput{Pipeline: "rerun"}}
	service := testService(pds, &testGit{}, &testKnot{capabilities: []string{"knot-repo-token-mint"}})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Spindle: optionalString("spindle.example")}}}
	service.spindle = testSpindleFactory{client: client}
	target := Target{Handle: "owner.test", Repo: "example"}
	approved, err := service.ApprovePipeline(context.Background(), target, "blocked-pipeline")
	if err != nil || !approved.ApprovalRequested || client.approveInput.Repo != "did:plc:repo" {
		t.Fatalf("approved=%+v err=%v input=%+v", approved, err, client.approveInput)
	}
	retry, err := service.RetryPipeline(context.Background(), target, "old-pipeline", []string{"test.yml"})
	if err != nil || retry.OriginalPipeline != "old-pipeline" || retry.Pipeline != "rerun" || !slices.Equal(client.retryInput.Workflows, []string{"test.yml"}) {
		t.Fatalf("retry=%+v err=%v input=%+v", retry, err, client.retryInput)
	}
	origin := service.knot.(*testKnotFactory).client.(*testKnot)
	if origin.procedureInputs[0]["lxm"] != "org.tangled.ci.approvePipeline" || origin.procedureInputs[1]["lxm"] != "org.tangled.ci.retryPipeline" {
		t.Fatalf("tokens=%+v", origin.procedureInputs)
	}
	client.err = errors.New("access denied")
	if _, err := service.ApprovePipeline(context.Background(), target, "blocked-pipeline"); err == nil {
		t.Fatal("approval denial hidden")
	}
	if _, err := service.RetryPipeline(context.Background(), target, "old-pipeline", nil); err == nil {
		t.Fatal("retry denial hidden")
	}
}

func TestBlockedWorkflowsCanBeCancelled(t *testing.T) {
	workflows := []spindle.Workflow{{Name: "build.yml", Status: "blocked"}, {Name: "done.yml", Status: "success"}}
	if err := validateSelectedWorkflows(workflows, []string{"build.yml"}); err != nil {
		t.Fatal(err)
	}
	if got := selectCancellableWorkflows(workflows, nil); !slices.Equal(got, []string{"build.yml"}) {
		t.Fatalf("cancellable=%v", got)
	}
}

func TestPipelineRevisionAcceptsBothObjectHashFormats(t *testing.T) {
	service := testService(nil, &testGit{}, nil)
	for _, commit := range []string{strings.Repeat("a", 40), strings.Repeat("b", 64)} {
		resolved, ref, err := service.resolvePipelineRevision(context.Background(), commit)
		if err != nil || ref != "" || resolved != commit {
			t.Fatalf("commit=%s resolved=%s ref=%s err=%v", commit, resolved, ref, err)
		}
	}
	for _, invalid := range []string{strings.Repeat("a", 41), strings.Repeat("z", 64)} {
		if isFullCommitSHA(invalid) {
			t.Fatalf("accepted invalid hash=%s", invalid)
		}
	}
}

func TestPipelineLogsPreserveStepOutcome(t *testing.T) {
	client := &testPipelineClient{
		pipeline:  &spindle.Pipeline{Repo: "did:plc:repo"},
		logEvents: []spindle.PipelineLogEvent{{Type: "control", Control: &spindle.PipelineLogControl{Status: "end", Outcome: "failed", Workflow: "ci.yml"}}},
	}
	service := testService(&testPDS{}, &testGit{}, &testKnot{})
	service.appview = testAppview{repo: &tangled.Repo{Value: tangled.RepoMetadata{Spindle: optionalString("spindle.example"), RepoDid: optionalString("did:plc:repo")}}}
	service.spindle = testSpindleFactory{client: client}
	var events []PipelineLogEvent
	err := service.PipelineLogs(context.Background(), Target{Handle: "owner.test", Repo: "example"}, "pipeline", nil, func(event PipelineLogEvent) error { events = append(events, event); return nil })
	if err != nil || len(events) != 1 || events[0].Control.Outcome != "failed" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}
