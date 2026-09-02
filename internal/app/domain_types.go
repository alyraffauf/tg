package app

// Author is the owner or creator of a record, resolved from a DID.
type Author struct {
	DID    string `json:"did"`
	Handle string `json:"handle"`
}

// RecordWarning describes a malformed or partially unavailable record that a
// list operation skipped while preserving its valid results.
type RecordWarning struct {
	URI   string `json:"uri,omitempty"`
	Error string `json:"error"`
}

type ItemListResult struct {
	Items    []Item          `json:"items"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

type RepoListResult struct {
	Items    []RepoItem      `json:"items"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

type StringListResult struct {
	Items    []StringItem    `json:"items"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

type SSHKeyListResult struct {
	Items    []SSHKeyItem    `json:"items"`
	Warnings []RecordWarning `json:"warnings,omitempty"`
}

// Item is a listing entry for an issue or a pull request. SourceBranch and
// TargetBranch are only populated (and only emitted as JSON) for pulls.
type Item struct {
	Rkey         string `json:"rkey"`
	URI          string `json:"uri"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Author       Author `json:"author"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	CommentCount int64  `json:"commentCount"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
}

// RepoItem is a single repository in a listing or view.
type RepoItem struct {
	Name        string    `json:"name"`
	URI         string    `json:"uri"`
	Author      string    `json:"author"`
	Knot        string    `json:"knot"`
	Description string    `json:"description,omitempty"`
	CreatedAt   string    `json:"createdAt"`
	RepoDid     string    `json:"repoDid,omitempty"`
	Stars       int64     `json:"stars,omitempty"`
	Pipeline    *Pipeline `json:"pipeline,omitempty"`
}

// Pipeline is one CI pipeline associated with a repository.
type Pipeline struct {
	ID         string             `json:"id"`
	Commit     string             `json:"commit"`
	CreatedAt  string             `json:"createdAt,omitempty"`
	Repo       string             `json:"repo,omitempty"`
	SourceRepo string             `json:"sourceRepo,omitempty"`
	Trigger    map[string]any     `json:"trigger"`
	Workflows  []PipelineWorkflow `json:"workflows"`
}

// PipelineWorkflow is one workflow executed by a pipeline.
type PipelineWorkflow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

// PipelineStatusResult is the status of a repository's latest pipeline.
type PipelineStatusResult struct {
	Commit      string    `json:"commit"`
	Pipeline    *Pipeline `json:"pipeline"`
	HasFailures bool      `json:"hasFailures"`
}

// PipelineCancelResult describes a cancelled pipeline or its selected workflows.
type PipelineCancelResult struct {
	Pipeline              string   `json:"pipeline"`
	Workflows             []string `json:"workflows,omitempty"`
	CancellationRequested bool     `json:"cancellationRequested"`
}

// PipelineTriggerResult describes a manually triggered pipeline.
type PipelineTriggerResult struct {
	Pipeline  string   `json:"pipeline"`
	Commit    string   `json:"commit"`
	Workflows []string `json:"workflows,omitempty"`
}

// SSHKeyItem is one SSH public key in a listing.
type SSHKeyItem struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
	URI       string `json:"uri"`
}

// StringItem is one tangled string in a listing.
type StringItem struct {
	Rkey        string `json:"rkey"`
	URI         string `json:"uri"`
	Filename    string `json:"filename"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// StringViewResult is the full view of a single string.
type StringViewResult struct {
	Rkey        string `json:"rkey"`
	URI         string `json:"uri"`
	Filename    string `json:"filename"`
	Author      Author `json:"author"`
	Description string `json:"description,omitempty"`
	Contents    string `json:"contents"`
	CreatedAt   string `json:"createdAt"`
}

// ViewResult is a single issue or pull request. SourceBranch and
// TargetBranch are only populated (and only emitted as JSON) for pulls.
type ViewResult struct {
	Rkey         string `json:"rkey"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Body         string `json:"body,omitempty"`
	Author       Author `json:"author"`
	CreatedAt    string `json:"createdAt"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
}

// CreatedRecordResult is returned by any operation that creates a record.
type CreatedRecordResult struct {
	Rkey string `json:"rkey"`
	URI  string `json:"uri"`
}

// DeletedRecordResult is returned by any operation that deletes a record.
type DeletedRecordResult struct {
	Rkey string `json:"rkey"`
}

// StateResult is returned by issue/PR state changes (close and reopen).
type StateResult struct {
	Rkey  string `json:"rkey"`
	State string `json:"state"`
}

// MergePullResult reports the two durable effects of merging a pull request.
// A merge cannot be undone when recording its ATProto status fails, so that
// failure is returned as a warning.
type MergePullResult struct {
	Rkey           string   `json:"rkey"`
	Merged         bool     `json:"merged"`
	StatusRecorded bool     `json:"statusRecorded"`
	Warnings       []string `json:"warnings,omitempty"`
}

// RepoCreateResult is returned by repository creation.
type RepoCreateResult struct {
	Handle        string   `json:"handle"`
	Name          string   `json:"name"`
	URI           string   `json:"uri"`
	Knot          string   `json:"knot"`
	Cloned        bool     `json:"cloned"`
	Pushed        bool     `json:"pushed"`
	DefaultBranch string   `json:"defaultBranch,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

// RepoCloneResult is returned by repository cloning.
type RepoCloneResult struct {
	Handle      string   `json:"handle"`
	Repo        string   `json:"repo"`
	RepoDID     string   `json:"repoDid,omitempty"`
	Destination string   `json:"destination"`
	Warnings    []string `json:"warnings,omitempty"`
}

// RepoEditResult is returned by repository edits.
type RepoEditResult struct {
	URI         string `json:"uri"`
	Description string `json:"description"`
}

// RepoDeleteResult is returned by repository deletion.
type RepoDeleteResult struct {
	URI string `json:"uri"`
}

// RepoDefaultBranchResult is returned by setting a repo's default branch.
type RepoDefaultBranchResult struct {
	URI    string `json:"uri"`
	Branch string `json:"branch"`
}

// RepoForkResult is returned by repository forking.
type RepoForkResult struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	URI    string `json:"uri"`
	Knot   string `json:"knot"`
}

// PRCreateResult is returned by pull request creation.
type PRCreateResult struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
	Base  string `json:"base"`
	Head  string `json:"head"`
}

// PRCheckoutResult is returned by pull request checkout.
type PRCheckoutResult struct {
	Rkey   string `json:"rkey"`
	Branch string `json:"branch"`
}

// SSHKeyAddResult is returned by SSH key addition.
type SSHKeyAddResult struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// AuthStatusResult is returned by an auth status probe.
type AuthStatusResult struct {
	Authenticated bool   `json:"authenticated"`
	Status        string `json:"status,omitempty"`
	DID           string `json:"did,omitempty"`
	Handle        string `json:"handle,omitempty"`
}

// AuthLogoutResult is returned by logout. WasLoggedIn reports whether a
// session existed and was cleared; it is false when there was nothing to
// log out (not a failure — the command still exits 0).
type AuthLogoutResult struct {
	WasLoggedIn bool `json:"wasLoggedIn"`
}

// AuthAccountResult is one account in an account listing or switch.
type AuthAccountResult struct {
	Active bool   `json:"active"`
	DID    string `json:"did"`
	Handle string `json:"handle"`
	Method string `json:"method"`
}

// GitCredentialResult contains credentials for a Git HTTPS request.
// MatchesRequestedHost is false when the request is not for the current
// repository's recorded Knot.
type GitCredentialResult struct {
	Token                string
	Handle               string
	MatchesRequestedHost bool
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
