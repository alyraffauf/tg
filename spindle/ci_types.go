package spindle

import (
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const (
	nsidQueryPipelines  syntax.NSID = "sh.tangled.ci.queryPipelines"
	nsidGetPipeline     syntax.NSID = "sh.tangled.ci.getPipeline"
	nsidCancelPipeline  syntax.NSID = "sh.tangled.ci.cancelPipeline"
	nsidTriggerPipeline syntax.NSID = "sh.tangled.ci.triggerPipeline"
	nsidSubscribeLogs   syntax.NSID = "sh.tangled.ci.subscribePipelineLogs"
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
	Repo      string        `json:"repo"`
	Trigger   ManualTrigger `json:"trigger"`
	Workflows []string      `json:"workflows,omitempty"`
}

type ManualTrigger struct {
	LexiconTypeID string `json:"$type"`
	SHA           string `json:"sha"`
	Ref           string `json:"ref,omitempty"`
}

type TriggerPipelineOutput struct {
	Pipeline string `json:"pipeline"`
}

// PipelineLogControl marks the start or end of a workflow step.
type PipelineLogControl struct {
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
