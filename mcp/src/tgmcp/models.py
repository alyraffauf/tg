"""Models for tg JSON output, matching internal/app/domain_types.go.

Timestamps and ATProto identifiers stay as strings to preserve CLI values.
Go nil slices and maps can appear as JSON null. Omitted scalar fields use
their Go zero values. Unknown fields are ignored for forward compatibility.
"""

from typing import Literal

from pydantic import BaseModel, ConfigDict, Field, RootModel
from pydantic.alias_generators import to_camel


class TGOutput(BaseModel):
    """Read Python field names or tg JSON keys. Write tg JSON keys."""

    model_config = ConfigDict(
        alias_generator=to_camel,
        populate_by_name=True,
        serialize_by_alias=True,
        strict=True,
    )


class Author(TGOutput):
    did: str
    handle: str


class RecordWarning(TGOutput):
    uri: str = ""
    error: str


class Item(TGOutput):
    """Issue or pull request listing entry."""

    rkey: str
    uri: str
    title: str
    state: str
    author: Author
    created_at: str
    updated_at: str = ""
    comment_count: int
    source_branch: str = ""
    target_branch: str = ""


class PipelineWorkflow(TGOutput):
    id: str
    name: str
    status: str
    error: str = ""
    started_at: str = ""
    finished_at: str = ""


class PipelineTrigger(TGOutput):
    """Pipeline trigger fields. Keep unknown fields from the spindle."""

    model_config = ConfigDict(extra="allow")

    trigger_type: str = Field(default="", alias="$type")
    ref: str = ""
    old_sha: str = ""
    new_sha: str = ""
    sha: str = ""
    pull: str = ""
    source_branch: str = ""
    target_branch: str = ""
    source_repo: str = ""
    source_sha: str = ""


class Pipeline(TGOutput):
    id: str
    commit: str
    created_at: str = ""
    repo: str = ""
    source_repo: str = ""
    trigger: PipelineTrigger | None
    workflows: list[PipelineWorkflow] | None


class RepoItem(TGOutput):
    name: str
    uri: str
    author: str
    knot: str
    description: str = ""
    created_at: str
    repo_did: str = ""
    stars: int = 0
    pipeline: Pipeline | None = None


class StringItem(TGOutput):
    rkey: str
    uri: str
    filename: str
    description: str = ""
    created_at: str


class SSHKeyItem(TGOutput):
    name: str
    key: str
    created_at: str
    uri: str


class ItemListResult(TGOutput):
    items: list[Item] | None
    warnings: list[RecordWarning] = Field(default_factory=list)


class RepoListResult(TGOutput):
    items: list[RepoItem] | None
    warnings: list[RecordWarning] = Field(default_factory=list)


class StringListResult(TGOutput):
    items: list[StringItem] | None
    warnings: list[RecordWarning] = Field(default_factory=list)


class SSHKeyListResult(TGOutput):
    items: list[SSHKeyItem] | None
    warnings: list[RecordWarning] = Field(default_factory=list)


class ViewResult(TGOutput):
    cid: str = ""
    uri: str = ""
    """Issue or pull request detail."""

    rkey: str
    title: str
    state: str
    body: str = ""
    author: Author
    created_at: str
    source_branch: str = ""
    target_branch: str = ""


class StringViewResult(TGOutput):
    rkey: str
    uri: str
    filename: str
    author: Author
    description: str = ""
    contents: str
    created_at: str


class CreatedRecordResult(TGOutput):
    cid: str = ""
    warnings: list[str] = Field(default_factory=list)
    rkey: str = Field(description="Record key for view or edit calls")
    uri: str = Field(description="ATProto record URI. The final component is the rkey.")


class DeletedRecordResult(TGOutput):
    rkey: str


class UpdatedRecordResult(TGOutput):
    cid: str = ""
    uri: str = ""
    warnings: list[str] = Field(default_factory=list)
    """Record key returned after an edit or pull request update."""

    rkey: str = Field(
        description="Updated record key. Use a view tool to read the result."
    )


class StateResult(TGOutput):
    cid: str = ""
    uri: str = ""
    warnings: list[str] = Field(default_factory=list)
    rkey: str
    state: str


class MergePullResult(TGOutput):
    """Merge result, including whether tg recorded the ATProto status."""

    rkey: str
    merged: bool = Field(description="True when the code landed on the target branch")
    status_recorded: bool = Field(
        description="Whether tg recorded the PR status. Check merged to see whether the code landed."
    )
    warnings: list[str] = Field(
        default_factory=list,
        description="Status recording errors after the code landed",
    )


class RepoCreateResult(TGOutput):
    handle: str
    name: str
    uri: str
    knot: str
    cloned: bool = Field(description="Whether the requested local clone succeeded")
    pushed: bool = Field(description="Whether the requested push succeeded")
    default_branch: str = ""
    warnings: list[str] = Field(
        default_factory=list,
        description="Registration or setup warnings. Check the repository before retrying.",
    )


class RepoCloneResult(TGOutput):
    handle: str
    repo: str
    repo_did: str = ""
    destination: str
    warnings: list[str] = Field(default_factory=list)


class RepoEditResult(TGOutput):
    warnings: list[str] = Field(default_factory=list)
    uri: str
    description: str


class RepoDeleteResult(TGOutput):
    uri: str


class RepoDefaultBranchResult(TGOutput):
    warnings: list[str] = Field(default_factory=list)
    uri: str
    branch: str


class RepoForkResult(TGOutput):
    import_status: str = ""
    import_reason: str = ""
    warnings: list[str] = Field(default_factory=list)
    handle: str
    name: str
    uri: str
    knot: str


class PRCreateResult(TGOutput):
    cid: str = ""
    warnings: list[str] = Field(default_factory=list)
    uri: str = Field(
        description="Created PR record URI. The final component is the rkey."
    )
    title: str
    base: str
    head: str


class PRCheckoutResult(TGOutput):
    rkey: str
    branch: str


class PullDiffResult(TGOutput):
    patch: str


class SSHKeyAddResult(TGOutput):
    name: str
    uri: str


class AuthStatusResult(TGOutput):
    authenticated: bool
    status: str = ""
    did: str = ""
    handle: str = ""


class AuthLogoutResult(TGOutput):
    was_logged_in: bool


class AuthAccountResult(TGOutput):
    active: bool
    did: str
    handle: str
    method: str


class AuthAccountListResult(RootModel[list[AuthAccountResult] | None]):
    """Top-level array returned by tg auth list."""


class PipelineListResult(RootModel[list[Pipeline] | None]):
    """Top-level array returned by tg pipeline list."""


class PipelineStatusResult(TGOutput):
    commit: str
    pipeline: Pipeline | None
    has_failures: bool


class PipelineCancelResult(TGOutput):
    pipeline: str
    workflows: list[str] = Field(default_factory=list)
    cancellation_requested: bool


class PipelineTriggerResult(TGOutput):
    pipeline: str
    pipelines: list[str] = Field(default_factory=list)
    message: str = ""
    commit: str
    workflows: list[str] = Field(default_factory=list)


class PipelineLogControl(TGOutput):
    outcome: str = ""
    kind: str
    step: int
    time: str
    status: str = ""
    content: str
    workflow: str
    command: str | None = None


class PipelineLogData(TGOutput):
    step: int
    time: str
    stream: str
    content: str
    workflow: str


class PipelineLogEvent(TGOutput):
    """One JSON line emitted by tg pipeline logs --json."""

    type: str
    control: PipelineLogControl | None = None
    data: PipelineLogData | None = None


class PipelineLogsResult(TGOutput):
    """Collected log events and why collection stopped."""

    events: list[PipelineLogEvent] = Field(default_factory=list)
    stop_reason: Literal[
        "complete", "timeout", "event_limit", "byte_limit", "error"
    ] = Field(
        description="Why collection stopped. complete means the stream ended successfully."
    )
    error: str | None = Field(
        default=None,
        description="Stream error. Collected events remain available.",
    )


class AttestationSubject(TGOutput):
    uri: str
    cid: str


class AttestationEntry(TGOutput):
    rkey: str
    subject: AttestationSubject
    change: str = ""
    created_at: str


class AttestationListResult(TGOutput):
    items: list[AttestationEntry]


class AttestationRetryResult(TGOutput):
    retried: int
    pending: int
    warnings: list[str] = Field(default_factory=list)


class PipelineApproveResult(TGOutput):
    pipeline: str
    approval_requested: bool


class PipelineRetryResult(TGOutput):
    pipeline: str
    original_pipeline: str
    workflows: list[str] = Field(default_factory=list)
