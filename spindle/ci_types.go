package spindle

import (
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const (
	nsidQueryPipelines  syntax.NSID = "org.tangled.ci.queryPipelines"
	nsidGetPipeline     syntax.NSID = "org.tangled.ci.getPipeline"
	nsidCancelPipeline  syntax.NSID = "org.tangled.ci.cancelPipeline"
	nsidNotifyEvent     syntax.NSID = "org.tangled.event.notifyEvent"
	nsidApprovePipeline syntax.NSID = "org.tangled.ci.approvePipeline"
	nsidRetryPipeline   syntax.NSID = "org.tangled.ci.retryPipeline"
	nsidSubscribeLogs   syntax.NSID = "org.tangled.ci.subscribePipelineLogs"
)

// Workflow is one workflow executed by a pipeline.
type Workflow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

// Pipeline is a CI pipeline returned by a spindle.
type Pipeline struct {
	ID         string         `json:"id"`
	Commit     string         `json:"commit"`
	CreatedAt  string         `json:"createdAt,omitempty"`
	Repo       string         `json:"repo,omitempty"`
	SourceRepo string         `json:"sourceRepo,omitempty"`
	Trigger    map[string]any `json:"trigger"`
	Workflows  []Workflow     `json:"workflows"`
}

type QueryPipelinesOutput struct {
	Pipelines []Pipeline `json:"pipelines"`
	Cursor    string     `json:"cursor,omitempty"`
	Total     int        `json:"total"`
}

type CancelPipelineInput struct {
	Pipeline  string   `json:"pipeline"`
	Repo      string   `json:"repo"`
	Workflows []string `json:"workflows,omitempty"`
}

type TriggerPipelineInput struct {
	Repo  string        `json:"repo"`
	Event ManualTrigger `json:"event"`
}

type ManualTrigger struct {
	LexiconTypeID string   `json:"$type"`
	SHA           string   `json:"commit"`
	Repo          string   `json:"repo"`
	Ref           string   `json:"ref,omitempty"`
	Workflows     []string `json:"workflows,omitempty"`
}

type ApprovePipelineInput struct {
	Repo     string `json:"repo"`
	Pipeline string `json:"pipeline"`
}

type RetryPipelineInput struct {
	Repo      string   `json:"repo"`
	Pipeline  string   `json:"pipeline"`
	Workflows []string `json:"workflows,omitempty"`
}

type RetryPipelineOutput struct {
	Pipeline string `json:"pipeline"`
}

type TriggerPipelineOutput struct {
	Pipelines []string `json:"pipelines"`
	Message   string   `json:"message,omitempty"`
}

// PipelineLogControl marks the start or end of a workflow step.
type PipelineLogControl struct {
	Outcome  string  `json:"outcome,omitempty"`
	Kind     string  `json:"kind"`
	Step     int64   `json:"step"`
	Time     string  `json:"time"`
	Status   string  `json:"status,omitempty"`
	Content  string  `json:"content"`
	Workflow string  `json:"workflow"`
	Command  *string `json:"command,omitempty"`
}

// PipelineLogData is one line of workflow output.
type PipelineLogData struct {
	Step     int64  `json:"step"`
	Time     string `json:"time"`
	Stream   string `json:"stream"`
	Content  string `json:"content"`
	Workflow string `json:"workflow"`
}

// PipelineLogEvent is one event from a log subscription.
type PipelineLogEvent struct {
	Type    string              `json:"type"`
	Control *PipelineLogControl `json:"control,omitempty"`
	Data    *PipelineLogData    `json:"data,omitempty"`
}
