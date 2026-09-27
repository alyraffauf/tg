"""MCP tools for the tg CLI."""

import csv
import io
import re
from typing import Annotated, Literal

from fastmcp import FastMCP
from fastmcp.exceptions import ToolError
from pydantic import Field

from tgmcp.models import (
    AuthAccountListResult,
    AuthStatusResult,
    CreatedRecordResult,
    DeletedRecordResult,
    ItemListResult,
    MergePullResult,
    Pipeline,
    PipelineCancelResult,
    PipelineListResult,
    PipelineLogsResult,
    PipelineTriggerResult,
    PRCheckoutResult,
    PRCreateResult,
    PullDiffResult,
    RepoCloneResult,
    RepoCreateResult,
    RepoDefaultBranchResult,
    RepoDeleteResult,
    RepoEditResult,
    RepoForkResult,
    RepoItem,
    RepoListResult,
    SSHKeyAddResult,
    SSHKeyListResult,
    StateResult,
    StringListResult,
    StringViewResult,
    UpdatedRecordResult,
    ViewResult,
)
from tgmcp.runner import TGConfig, TGRunner

mcp = FastMCP(
    "tgmcp",
    instructions="""Tools for Tangled, a Git forge on ATProto. Use them to manage
repositories, issues, and pull requests, inspect patches, and monitor CI pipelines.""",
)
runner = TGRunner(TGConfig.from_env())
READ_ONLY = {"readOnlyHint": True, "idempotentHint": True}
WRITE = {
    "readOnlyHint": False,
    "destructiveHint": False,
    "idempotentHint": False,
}

type RepoTarget = Annotated[
    str,
    Field(
        description="Tangled repository as handle/repo, such as aly.codes/tg",
        pattern=r"^[^\s/]+/[^\s/]+$",
    ),
]

type RecordKey = Annotated[
    str,
    Field(
        description="Record key, the final component of the record's at:// URI",
        pattern=r"^[^\s/]+$",
    ),
]
type Account = Annotated[str, Field(description="Account handle or DID", min_length=1)]
type ResultLimit = Annotated[int, Field(gt=0)]
type SortOrder = Literal["asc", "desc"]
type RecordTitle = Annotated[
    str, Field(description="Issue or pull request title", pattern=r"\S")
]
type CommentBody = Annotated[
    str, Field(description="Comment text in Markdown", pattern=r"\S")
]


@mcp.tool(annotations=READ_ONLY)
async def view_repo(repo: RepoTarget) -> RepoItem:
    """Get repository metadata and its latest pipeline, if available."""
    return await runner.run_json(["repo", "view", "--", repo], RepoItem)


@mcp.tool(annotations=READ_ONLY)
async def list_repos(account: Account | None = None) -> RepoListResult:
    """List an account's repositories. Uses the tg account when omitted."""
    arguments = ["repo", "list", "--"]
    if account is not None:
        arguments.append(account)
    return await runner.run_json(arguments, RepoListResult)


@mcp.tool(annotations=READ_ONLY)
async def search_repos(
    query: Annotated[str, Field(min_length=1)], limit: ResultLimit = 50
) -> RepoListResult:
    """Search repositories indexed by Tangled."""
    return await runner.run_json(
        ["repo", "search", "--limit", str(limit), "--", query], RepoListResult
    )


def item_list_arguments(
    kind: Literal["issue", "pr"],
    repo: str,
    *,
    author: str | None,
    state: str | None,
    limit: int,
    order: SortOrder | None,
) -> list[str]:
    arguments = [kind, "list", "--limit", str(limit)]
    for flag, value in (("author", author), ("state", state), ("order", order)):
        if value is not None:
            arguments.extend([f"--{flag}", value])
    return [*arguments, "--", repo]


@mcp.tool(annotations=READ_ONLY)
async def list_issues(
    repo: RepoTarget,
    author: Account | None = None,
    state: Literal["open", "closed"] | None = None,
    limit: ResultLimit = 50,
    order: SortOrder | None = None,
) -> ItemListResult:
    """List issues and their record keys. Filter by author or state."""
    arguments = item_list_arguments(
        "issue", repo, author=author, state=state, limit=limit, order=order
    )
    return await runner.run_json(arguments, ItemListResult)


@mcp.tool(annotations=READ_ONLY)
async def view_issue(repo: RepoTarget, rkey: RecordKey) -> ViewResult:
    """Read an issue's body and state using an rkey from list_issues."""
    return await runner.run_json(
        ["issue", "view", "--repo", repo, "--", rkey], ViewResult
    )


@mcp.tool(annotations=WRITE)
async def create_issue(
    repo: RepoTarget, title: RecordTitle, body: str = ""
) -> CreatedRecordResult:
    """Create an issue. The body supports Markdown.

    After a timeout, check whether the issue exists before retrying.
    """
    return await runner.run_json(
        ["issue", "create", "--repo", repo, f"--body={body}", "--", title],
        CreatedRecordResult,
    )


@mcp.tool(annotations={**WRITE, "idempotentHint": True})
async def edit_issue(
    rkey: RecordKey, title: RecordTitle | None = None, body: str | None = None
) -> UpdatedRecordResult:
    """Edit an issue owned by the tg account.

    Provide a title or body. Omitted or null fields stay unchanged.
    Set body to an empty string to clear it.
    """
    if title is None and body is None:
        raise ToolError("Provide title or body to edit an issue")
    arguments = ["issue", "edit"]
    if title is not None:
        arguments.append(f"--title={title}")
    if body is not None:
        arguments.append(f"--body={body}")
    return await runner.run_json([*arguments, "--", rkey], UpdatedRecordResult)


@mcp.tool(annotations=WRITE)
async def comment_on_issue(
    repo: RepoTarget, rkey: RecordKey, body: CommentBody
) -> CreatedRecordResult:
    """Comment on an issue. The body supports Markdown.

    After a timeout, check whether the comment exists before retrying.
    """
    return await runner.run_json(
        ["issue", "comment", "--repo", repo, f"--body={body}", "--", rkey],
        CreatedRecordResult,
    )


@mcp.tool(annotations=WRITE)
async def close_issue(repo: RepoTarget, rkey: RecordKey) -> StateResult:
    """Close an issue."""
    return await runner.run_json(
        ["issue", "close", "--repo", repo, "--", rkey], StateResult
    )


@mcp.tool(annotations=WRITE)
async def reopen_issue(repo: RepoTarget, rkey: RecordKey) -> StateResult:
    """Reopen a closed issue."""
    return await runner.run_json(
        ["issue", "reopen", "--repo", repo, "--", rkey], StateResult
    )


@mcp.tool(annotations=READ_ONLY)
async def list_pull_requests(
    repo: RepoTarget,
    author: Account | None = None,
    state: Literal["open", "closed", "merged"] | None = None,
    limit: ResultLimit = 50,
    order: SortOrder | None = None,
) -> ItemListResult:
    """List pull requests and their record keys. Filter by author or state."""
    arguments = item_list_arguments(
        "pr", repo, author=author, state=state, limit=limit, order=order
    )
    return await runner.run_json(arguments, ItemListResult)


@mcp.tool(annotations=READ_ONLY)
async def view_pull_request(repo: RepoTarget, rkey: RecordKey) -> ViewResult:
    """Read a pull request using an rkey from list_pull_requests. Use diff_pull_request for its patch."""
    return await runner.run_json(["pr", "view", "--repo", repo, "--", rkey], ViewResult)


@mcp.tool(annotations=READ_ONLY)
async def diff_pull_request(repo: RepoTarget, rkey: RecordKey) -> PullDiffResult:
    """Read the latest pull request patch for code review."""
    return await runner.run_json(
        ["pr", "diff", "--repo", repo, "--", rkey], PullDiffResult
    )


@mcp.tool(annotations={**WRITE, "idempotentHint": True})
async def edit_pull_request(
    rkey: RecordKey, title: RecordTitle | None = None, body: str | None = None
) -> UpdatedRecordResult:
    """Change the title or body of a pull request authored by the tg account.

    Null leaves a field unchanged. An empty body clears it.
    Use update_pull_request to submit code changes.
    """
    if title is None and body is None:
        raise ToolError("Provide title or body to edit a pull request")
    arguments = ["pr", "edit"]
    if title is not None:
        arguments.append(f"--title={title}")
    if body is not None:
        arguments.append(f"--body={body}")
    return await runner.run_json([*arguments, "--", rkey], UpdatedRecordResult)


@mcp.tool(annotations=WRITE)
async def comment_on_pull_request(
    repo: RepoTarget, rkey: RecordKey, body: CommentBody
) -> CreatedRecordResult:
    """Add a Markdown comment to a pull request."""
    return await runner.run_json(
        ["pr", "comment", "--repo", repo, f"--body={body}", "--", rkey],
        CreatedRecordResult,
    )


@mcp.tool(annotations=WRITE)
async def close_pull_request(repo: RepoTarget, rkey: RecordKey) -> StateResult:
    """Mark a pull request as closed."""
    return await runner.run_json(
        ["pr", "close", "--repo", repo, "--", rkey], StateResult
    )


@mcp.tool(annotations=WRITE)
async def reopen_pull_request(repo: RepoTarget, rkey: RecordKey) -> StateResult:
    """Reopen a closed pull request."""
    return await runner.run_json(
        ["pr", "reopen", "--repo", repo, "--", rkey], StateResult
    )


@mcp.tool(annotations={**WRITE, "destructiveHint": True})
async def merge_pull_request(repo: RepoTarget, rkey: RecordKey) -> MergePullResult:
    """Merge a pull request into its target branch.

    merged=true means the code landed, even if statusRecorded=false.
    Check warnings for a status error. Check the branch after a timeout.
    """
    return await runner.run_json(
        ["pr", "merge", "--repo", repo, "--", rkey], MergePullResult
    )


@mcp.tool(annotations=READ_ONLY)
async def list_strings(account: Account | None = None) -> StringListResult:
    """List an account's strings. Uses the tg account when omitted."""
    arguments = ["string", "list", "--"]
    if account is not None:
        arguments.append(account)
    return await runner.run_json(arguments, StringListResult)


@mcp.tool(annotations=READ_ONLY)
async def view_string(
    rkey: RecordKey, account: Account | None = None
) -> StringViewResult:
    """Read a string. Uses the tg account when omitted."""
    arguments = ["string", "view", "--", rkey]
    if account is not None:
        arguments.append(account)
    return await runner.run_json(arguments, StringViewResult)


@mcp.tool(annotations=READ_ONLY)
async def list_ssh_keys(account: Account | None = None) -> SSHKeyListResult:
    """List public SSH keys. Uses the tg account when omitted."""
    arguments = ["ssh-key", "list", "--"]
    if account is not None:
        arguments.append(account)
    return await runner.run_json(arguments, SSHKeyListResult)


@mcp.tool(annotations=READ_ONLY)
async def list_pipelines(repo: RepoTarget) -> PipelineListResult:
    """Find pipeline IDs for view_pipeline, pipeline_logs, or cancel_pipeline."""
    return await runner.run_json(["pipeline", "list", "--", repo], PipelineListResult)


@mcp.tool(annotations=READ_ONLY)
async def view_pipeline(
    repo: RepoTarget, pipeline_id: Annotated[str, Field(min_length=1)]
) -> Pipeline:
    """Get a pipeline and its workflow results."""
    return await runner.run_json(
        ["pipeline", "view", "--repo", repo, "--", pipeline_id], Pipeline
    )


@mcp.tool(annotations=READ_ONLY)
async def pipeline_status(repo: RepoTarget) -> Pipeline:
    """Get the default branch's latest pipeline. Failed workflows are included."""
    # tg emits valid JSON and silently exits 1 when workflows failed.
    return await runner.run_json(
        ["pipeline", "status", "--", repo],
        Pipeline,
        expected_failure=(1, ""),
    )


@mcp.tool(annotations=READ_ONLY)
async def auth_status() -> AuthStatusResult:
    """Check whether the tg account is logged in. Ask the user to run tg auth login <handle> if needed."""
    return await runner.run_json(["auth", "status"], AuthStatusResult)


@mcp.tool(annotations=READ_ONLY)
async def list_accounts() -> AuthAccountListResult:
    """List accounts saved in tg."""
    return await runner.run_json(["auth", "list"], AuthAccountListResult)


type AbsolutePath = Annotated[
    str, Field(description="Absolute path on the server's filesystem", pattern=r"^/")
]
type NonEmptyText = Annotated[str, Field(min_length=1)]


def optional_flags(**values: str | int | None) -> list[str]:
    return [
        f"--{name.replace('_', '-')}={value}"
        for name, value in values.items()
        if value is not None
    ]


def list_flag(name: str, values: list[str] | None) -> list[str]:
    if not values:
        return []
    buffer = io.StringIO()
    csv.writer(buffer).writerow(values)
    return [f"--{name}={buffer.getvalue()[:-2]}"]


@mcp.tool(annotations=WRITE)
async def clone_repo(repo: RepoTarget, destination: AbsolutePath) -> RepoCloneResult:
    """Clone a repository to an absolute path on the server. Check warnings for setup problems."""
    return await runner.run_json(
        ["repo", "clone", "--", repo, destination], RepoCloneResult
    )


@mcp.tool(annotations=WRITE)
async def create_repo(
    name: RecordKey,
    description: str = "",
    knot: NonEmptyText | None = None,
    clone: bool = False,
    working_directory: AbsolutePath | None = None,
    push_path: AbsolutePath | None = None,
    remote: NonEmptyText = "origin",
    ssh_port: Annotated[int, Field(ge=1, le=65535)] | None = None,
) -> RepoCreateResult:
    """Create a repository under the tg account.

    clone creates a checkout inside working_directory.
    push_path uploads an existing local repository.
    Check cloned, pushed, and warnings for the setup result.
    """
    if clone and working_directory is None:
        raise ToolError(
            "Provide an absolute working_directory where tg can clone the new repository"
        )
    arguments = [
        "repo",
        "create",
        *optional_flags(
            description=description,
            knot=knot,
            push=push_path,
            remote=remote,
            ssh_port=ssh_port,
        ),
    ]
    if clone:
        arguments.append("--clone")
    return await runner.run_json(
        [*arguments, "--", name], RepoCreateResult, working_directory=working_directory
    )


@mcp.tool(annotations={**WRITE, "destructiveHint": True})
async def delete_repo(repo: RepoTarget) -> RepoDeleteResult:
    """Permanently delete a repository and its Git data."""
    return await runner.run_json(
        ["repo", "delete", "--yes", "--", repo], RepoDeleteResult
    )


@mcp.tool(annotations={**WRITE, "idempotentHint": True})
async def edit_repo(
    repo: RepoTarget,
    description: str | None = None,
    website: str | None = None,
    spindle: str | None = None,
    add_labels: list[NonEmptyText] | None = None,
    remove_labels: list[NonEmptyText] | None = None,
) -> RepoEditResult:
    """Change repository metadata or labels.

    Null leaves a field unchanged. An empty string clears it.
    """
    flags = optional_flags(description=description, website=website, spindle=spindle)
    flags += list_flag("add-label", add_labels) + list_flag(
        "remove-label", remove_labels
    )
    if not flags:
        raise ToolError("Provide a repository field or label to update")
    return await runner.run_json(["repo", "edit", *flags, "--", repo], RepoEditResult)


@mcp.tool(annotations=WRITE)
async def fork_repo(
    repo: RepoTarget, name: RecordKey | None = None, knot: NonEmptyText | None = None
) -> RepoForkResult:
    """Fork a repository under the tg account."""
    arguments = ["repo", "fork", *optional_flags(knot=knot), "--", repo]
    if name is not None:
        arguments.append(name)
    return await runner.run_json(arguments, RepoForkResult)


@mcp.tool(annotations={**WRITE, "idempotentHint": True})
async def set_repo_default_branch(
    repo: RepoTarget, branch: NonEmptyText
) -> RepoDefaultBranchResult:
    """Set a repository's default branch."""
    return await runner.run_json(
        ["repo", "set-default-branch", "--", branch, repo], RepoDefaultBranchResult
    )


@mcp.tool(annotations=WRITE)
async def create_pull_request(
    repo: RepoTarget,
    working_directory: AbsolutePath,
    title: RecordTitle,
    body: str = "",
    base: NonEmptyText | None = None,
    head: NonEmptyText | None = None,
    source_repo: RepoTarget | None = None,
) -> PRCreateResult:
    """Create a pull request from a local checkout.

    head defaults to the current branch. For a fork, provide source_repo and base.
    Use an existing checkout as working_directory.
    """
    arguments = [
        "pr",
        "create",
        *optional_flags(
            repo=repo,
            title=title,
            body=body,
            base=base,
            head=head,
            source_repo=source_repo,
        ),
    ]
    return await runner.run_json(
        arguments, PRCreateResult, working_directory=working_directory
    )


@mcp.tool(annotations=WRITE)
async def update_pull_request(
    rkey: RecordKey, working_directory: AbsolutePath, base: NonEmptyText | None = None
) -> UpdatedRecordResult:
    """Submit code changes as a new patch round for a PR authored by the tg account.

    Use the source checkout as working_directory.
    Use edit_pull_request for title or body changes.
    """
    arguments = ["pr", "update", *optional_flags(base=base), "--", rkey]
    return await runner.run_json(
        arguments, UpdatedRecordResult, working_directory=working_directory
    )


@mcp.tool(annotations={**WRITE, "destructiveHint": True})
async def checkout_pull_request(
    repo: RepoTarget,
    rkey: RecordKey,
    working_directory: AbsolutePath,
    branch: NonEmptyText | None = None,
    force: bool = False,
) -> PRCheckoutResult:
    """Apply a pull request's latest patch to a local branch.

    force resets an existing checkout branch.
    Use a checkout of the target repository as working_directory.
    Use diff_pull_request to review the patch without applying it.
    """
    arguments = ["pr", "checkout", *optional_flags(repo=repo, branch=branch)]
    if force:
        arguments.append("--force")
    return await runner.run_json(
        [*arguments, "--", rkey], PRCheckoutResult, working_directory=working_directory
    )


@mcp.tool(annotations=WRITE)
async def trigger_pipeline(
    repo: RepoTarget,
    revision: NonEmptyText,
    workflows: list[NonEmptyText] | None = None,
    working_directory: AbsolutePath | None = None,
) -> PipelineTriggerResult:
    """Start CI for a commit or branch. Branch names need a local checkout."""
    if working_directory is None and re.fullmatch(r"[0-9a-fA-F]{40}", revision) is None:
        raise ToolError(
            "Provide an absolute working_directory containing a checkout to resolve "
            "the revision, or pass a full 40-character commit SHA"
        )
    arguments = [
        "pipeline",
        "trigger",
        *optional_flags(repo=repo),
        *list_flag("workflow", workflows),
        "--",
        revision,
    ]
    return await runner.run_json(
        arguments, PipelineTriggerResult, working_directory=working_directory
    )


@mcp.tool(annotations=WRITE)
async def cancel_pipeline(
    repo: RepoTarget,
    pipeline_id: NonEmptyText,
    workflows: list[NonEmptyText] | None = None,
) -> PipelineCancelResult:
    """Cancel pending or running workflows. Omit workflows to cancel all."""
    arguments = [
        "pipeline",
        "cancel",
        *optional_flags(repo=repo),
        *list_flag("workflow", workflows),
        "--",
        pipeline_id,
    ]
    return await runner.run_json(arguments, PipelineCancelResult)


@mcp.tool(annotations=READ_ONLY)
async def pipeline_logs(
    repo: RepoTarget,
    pipeline_id: NonEmptyText | None = None,
    workflows: list[NonEmptyText] | None = None,
    max_events: Annotated[int, Field(ge=1, le=10000)] = 500,
    max_bytes: Annotated[int, Field(ge=1, le=1048576)] = 262144,
    timeout_seconds: Annotated[float, Field(gt=0, le=60, allow_inf_nan=False)] = 10,
) -> PipelineLogsResult:
    """Read pipeline logs within time and output limits.

    Omit pipeline_id for the latest pipeline. stopReason explains why collection ended.
    Earlier events remain available if the stream fails. The server timeout also applies.
    A limit or timeout can leave logs incomplete. Filter workflows to narrow the output.
    """
    arguments = [
        "pipeline",
        "logs",
        *optional_flags(repo=repo),
        *list_flag("workflow", workflows),
        "--",
    ]
    if pipeline_id is not None:
        arguments.append(pipeline_id)
    return await runner.collect_logs(
        arguments,
        max_events=max_events,
        max_bytes=max_bytes,
        timeout_seconds=timeout_seconds,
    )


@mcp.tool(annotations=WRITE)
async def create_string(
    filename: NonEmptyText,
    contents: Annotated[str, Field(min_length=1)],
    description: str = "",
) -> CreatedRecordResult:
    """Publish a Tangled string. Pass a display filename and the text as contents."""
    arguments = [
        "string",
        "create",
        *optional_flags(filename=filename, description=description),
        "--",
        "-",
    ]
    return await runner.run_json(
        arguments, CreatedRecordResult, input_data=contents.encode("utf-8")
    )


@mcp.tool(annotations={**WRITE, "destructiveHint": True})
async def delete_string(rkey: RecordKey) -> DeletedRecordResult:
    """Delete a string owned by the tg account."""
    return await runner.run_json(["string", "delete", "--", rkey], DeletedRecordResult)


@mcp.tool(annotations=WRITE)
async def add_ssh_key(
    key_file: AbsolutePath, title: NonEmptyText | None = None
) -> SSHKeyAddResult:
    """Register a public SSH key from an absolute file path on the server."""
    return await runner.run_json(
        ["ssh-key", "add", *optional_flags(title=title), "--", key_file],
        SSHKeyAddResult,
    )


@mcp.tool(annotations={**WRITE, "destructiveHint": True})
async def delete_ssh_key(rkey: RecordKey) -> DeletedRecordResult:
    """Remove a public SSH key from the tg account."""
    return await runner.run_json(["ssh-key", "delete", "--", rkey], DeletedRecordResult)
