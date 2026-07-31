package spindle

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
