"""Tool behavior through the MCP protocol."""

import csv
import io
import json
from pathlib import Path
from typing import Any

import pytest
from fastmcp import Client
from fastmcp.exceptions import ToolError

from tests.conftest import FakeTG


async def test_create_issue_preserves_literal_input(
    client: Client, fake_tg: FakeTG
) -> None:
    fake_tg.configure(stdout='{"rkey":"key","uri":"at://example"}')
    result = await client.call_tool(
        "create_issue",
        {
            "repo": "alice.example/repo",
            "title": "--literal $(touch nope)",
            "body": "first\nsecond",
        },
    )
    assert result.structured_content == {
        "rkey": "key",
        "uri": "at://example",
        "cid": "",
        "warnings": [],
    }
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        "issue",
        "create",
        "--repo",
        "alice.example/repo",
        "--body=first\nsecond",
        "--",
        "--literal $(touch nope)",
    ]
    assert len(fake_tg.requests) == 1


@pytest.mark.parametrize("tool", ["edit_issue", "edit_pull_request"])
@pytest.mark.parametrize(
    "fields, expected",
    [
        ({"title": "new"}, ["--title=new"]),
        ({"body": ""}, ["--body="]),
        ({"title": None, "body": "new"}, ["--body=new"]),
    ],
)
async def test_edit_field_semantics(
    client: Client,
    fake_tg: FakeTG,
    tool: str,
    fields: dict[str, Any],
    expected: list[str],
) -> None:
    result = await client.call_tool(
        tool, {"repo": "alice.example/repo", "rkey": "key", **fields}
    )
    assert result.structured_content == {
        "rkey": "key",
        "warnings": [],
        "uri": "",
        "cid": "",
    }
    kind = "issue" if tool == "edit_issue" else "pr"
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        kind,
        "edit",
        "--repo",
        "alice.example/repo",
        *expected,
        "--",
        "key",
    ]


@pytest.mark.parametrize(
    "tool, kind", [("list_issues", "issue"), ("list_pull_requests", "pr")]
)
async def test_list_labels_filter_repeats_exact_names(
    client: Client, fake_tg: FakeTG, tool: str, kind: str
) -> None:
    labelled = {
        "rkey": "one",
        "uri": "at://example/one",
        "title": "First",
        "state": "open",
        "author": {"did": "did:plc:alice", "handle": "alice.example"},
        "createdAt": "2026-01-01T00:00:00Z",
        "commentCount": 2,
        "labels": [
            {"name": "bug", "color": "#d73a4a", "uri": "at://example/labels/bug"},
            {"name": "help wanted", "uri": "at://example/labels/help"},
        ],
    }
    unlabelled = {**labelled, "rkey": "two", "uri": "at://example/two"}
    del unlabelled["labels"]
    fake_tg.configure(stdout=json.dumps({"items": [labelled, unlabelled]}))
    result = await client.call_tool(
        tool,
        {
            "repo": "alice.example/repo",
            "author": "alice.example",
            "state": "open",
            "labels": ["bug", "help wanted"],
            "limit": 5,
            "order": "asc",
        },
    )
    assert result.structured_content["items"][0]["labels"] == [
        {"name": "bug", "color": "#d73a4a", "uri": "at://example/labels/bug"},
        {"name": "help wanted", "color": "", "uri": "at://example/labels/help"},
    ]
    assert result.structured_content["items"][1]["labels"] == []
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        kind,
        "list",
        "--limit",
        "5",
        "--author",
        "alice.example",
        "--state",
        "open",
        "--order",
        "asc",
        "--label=bug",
        "--label=help wanted",
        "--",
        "alice.example/repo",
    ]


async def test_view_repo_lists_label_definitions(
    client: Client, fake_tg: FakeTG
) -> None:
    fake_tg.configure(
        stdout=json.dumps(
            {
                "name": "repo",
                "uri": "at://example/repo",
                "author": "alice.example",
                "knot": "knot.example",
                "createdAt": "2026-01-01T00:00:00Z",
                "labels": [{"name": "bug", "uri": "at://example/labels/bug"}],
                "warnings": ["label definitions unavailable"],
            }
        )
    )
    result = await client.call_tool("view_repo", {"repo": "alice.example/repo"})
    assert result.structured_content["labels"] == [
        {"name": "bug", "color": "", "uri": "at://example/labels/bug"}
    ]
    assert result.structured_content["warnings"] == ["label definitions unavailable"]
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        "repo",
        "view",
        "--",
        "alice.example/repo",
    ]


@pytest.mark.parametrize(
    "tool, kind", [("view_issue", "issue"), ("view_pull_request", "pr")]
)
async def test_view_item_returns_labels(
    client: Client, fake_tg: FakeTG, tool: str, kind: str
) -> None:
    fake_tg.configure(
        stdout=json.dumps(
            {
                "rkey": "key",
                "title": "Title",
                "state": "open",
                "author": {"did": "did:plc:alice", "handle": "alice.example"},
                "createdAt": "2026-01-01T00:00:00Z",
                "labels": [
                    {
                        "name": "",
                        "uri": "at://example/labels/missing",
                    }
                ],
                "warnings": ["unresolved label at://example/labels/missing"],
            }
        )
    )
    result = await client.call_tool(tool, {"repo": "alice.example/repo", "rkey": "key"})
    assert result.structured_content["labels"] == [
        {"name": "", "color": "", "uri": "at://example/labels/missing"}
    ]
    assert result.structured_content["warnings"] == [
        "unresolved label at://example/labels/missing"
    ]
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        kind,
        "view",
        "--repo",
        "alice.example/repo",
        "--",
        "key",
    ]


@pytest.mark.parametrize(
    "tool, kind", [("edit_issue", "issue"), ("edit_pull_request", "pr")]
)
async def test_label_only_edit_is_allowed(
    client: Client, fake_tg: FakeTG, tool: str, kind: str
) -> None:
    await client.call_tool(
        tool,
        {
            "repo": "alice.example/repo",
            "rkey": "key",
            "add_labels": ["bug", "one,two"],
            "remove_labels": ["wontfix"],
        },
    )
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        kind,
        "edit",
        "--repo",
        "alice.example/repo",
        "--add-label=bug",
        "--add-label=one,two",
        "--remove-label=wontfix",
        "--",
        "key",
    ]
    await client.call_tool(
        tool,
        {
            "repo": "alice.example/repo",
            "rkey": "key",
            "title": "New",
            "add_labels": ["bug"],
        },
    )
    assert fake_tg.requests[1]["arguments"] == [
        "--json",
        kind,
        "edit",
        "--repo",
        "alice.example/repo",
        "--title=New",
        "--add-label=bug",
        "--",
        "key",
    ]


@pytest.mark.parametrize("tool", ["edit_issue", "edit_pull_request"])
@pytest.mark.parametrize(
    "fields",
    [
        {"add_labels": ["bug"], "remove_labels": ["other", "bug"]},
        {"add_labels": [], "remove_labels": []},
        {"add_labels": []},
    ],
)
async def test_label_edit_without_a_change_does_not_launch_tg(
    client: Client, fake_tg: FakeTG, tool: str, fields: dict[str, Any]
) -> None:
    with pytest.raises(ToolError):
        await client.call_tool(
            tool, {"repo": "alice.example/repo", "rkey": "key", **fields}
        )
    assert not fake_tg.requests


@pytest.mark.parametrize(
    "tool, arguments",
    [
        ("edit_issue", {"rkey": "key"}),
        ("edit_pull_request", {"rkey": "key", "title": None, "body": None}),
        ("create_issue", {"repo": "alice.example/repo", "title": "  "}),
        ("view_repo", {"repo": "not-a-target"}),
        (
            "checkout_pull_request",
            {
                "repo": "alice.example/repo",
                "rkey": "key",
                "working_directory": "relative",
            },
        ),
        ("trigger_pipeline", {"repo": "alice.example/repo", "revision": "main"}),
    ],
)
async def test_invalid_inputs_do_not_launch_tg(
    client: Client, fake_tg: FakeTG, tool: str, arguments: dict[str, Any]
) -> None:
    with pytest.raises(ToolError):
        await client.call_tool(tool, arguments)
    assert not fake_tg.requests


async def test_string_uses_utf8_stdin(client: Client, fake_tg: FakeTG) -> None:
    fake_tg.configure(stdout='{"rkey":"key","uri":"at://example"}')
    await client.call_tool(
        "create_string", {"filename": "hello.txt", "contents": "héllo\n世界"}
    )
    request = fake_tg.requests[0]
    assert request["stdin"] == "héllo\n世界"
    assert request["arguments"] == [
        "--json",
        "string",
        "create",
        "--filename=hello.txt",
        "--description=",
        "--",
        "-",
    ]


async def test_git_tool_uses_cwd(
    client: Client, fake_tg: FakeTG, tmp_path: Path
) -> None:
    await client.call_tool(
        "update_pull_request",
        {
            "repo": "alice.example/repo",
            "rkey": "key",
            "working_directory": str(tmp_path),
            "base": "main",
        },
    )
    assert fake_tg.requests[0]["cwd"] == str(tmp_path)
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        "pr",
        "update",
        "--repo=alice.example/repo",
        "--base=main",
        "--",
        "key",
    ]


@pytest.mark.parametrize(
    "tool, arguments, flag, values, payload",
    [
        (
            "edit_repo",
            {"repo": "alice.example/repo", "add_labels": ["one,two", 'say "hi"']},
            "add-label",
            ["one,two", 'say "hi"'],
            {"uri": "at://example", "description": ""},
        ),
        (
            "cancel_pipeline",
            {
                "repo": "alice.example/repo",
                "pipeline_id": "pipeline",
                "workflows": ["build,docs", "test"],
            },
            "workflow",
            ["build,docs", "test"],
            {"pipeline": "pipeline", "cancellationRequested": True},
        ),
    ],
)
async def test_list_flags_preserve_values(
    client: Client,
    fake_tg: FakeTG,
    tool: str,
    arguments: dict[str, Any],
    flag: str,
    values: list[str],
    payload: dict[str, Any],
) -> None:
    fake_tg.configure(stdout=json.dumps(payload))
    await client.call_tool(tool, arguments)
    encoded = next(
        arg.split("=", 1)[1]
        for arg in fake_tg.requests[0]["arguments"]
        if arg.startswith(f"--{flag}=")
    )
    assert next(csv.reader(io.StringIO(encoded))) == values


async def test_delete_repo_confirms_cli_deletion(
    client: Client, fake_tg: FakeTG
) -> None:
    fake_tg.configure(stdout='{"uri":"at://example"}')
    await client.call_tool("delete_repo", {"repo": "alice.example/repo"})
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        "repo",
        "delete",
        "--yes",
        "--",
        "alice.example/repo",
    ]


@pytest.mark.parametrize(
    "code, stderr, valid",
    [(0, "", True), (1, "", True), (2, "", False), (1, "network failed", False)],
)
async def test_failed_pipeline_exit_contract(
    client: Client,
    fake_tg: FakeTG,
    pipeline_json: str,
    code: int,
    stderr: str,
    valid: bool,
) -> None:
    fake_tg.configure(stdout=pipeline_json, exit_code=code, stderr=stderr)
    arguments = {"repo": "alice.example/repo"}
    if valid:
        result = await client.call_tool("pipeline_status", arguments)
        assert result.structured_content is not None
        assert result.structured_content["workflows"][0]["status"] == "failed"
        assert (
            result.structured_content["trigger"]["$type"]
            == "sh.tangled.ci.trigger#push"
        )
    else:
        with pytest.raises(ToolError):
            await client.call_tool("pipeline_status", arguments)


async def test_silent_failure_still_requires_valid_json(
    client: Client, fake_tg: FakeTG
) -> None:
    fake_tg.configure(stdout="{}", exit_code=1)
    with pytest.raises(ToolError, match="did not match Pipeline"):
        await client.call_tool("pipeline_status", {"repo": "alice.example/repo"})


async def test_partial_merge_is_success_with_warning(
    client: Client, fake_tg: FakeTG
) -> None:
    payload = {
        "rkey": "key",
        "merged": True,
        "statusRecorded": False,
        "warnings": ["status write failed"],
    }
    fake_tg.configure(stdout=json.dumps(payload))
    result = await client.call_tool(
        "merge_pull_request", {"repo": "alice.example/repo", "rkey": "key"}
    )
    assert result.structured_content == payload
    assert len(fake_tg.requests) == 1


async def test_write_failure_is_not_retried(client: Client, fake_tg: FakeTG) -> None:
    fake_tg.configure(stderr="permission denied", exit_code=1)
    with pytest.raises(ToolError, match="permission denied"):
        await client.call_tool(
            "create_issue", {"repo": "alice.example/repo", "title": "new"}
        )
    assert len(fake_tg.requests) == 1


def check_schema(schema: Any, path: str) -> None:
    if not isinstance(schema, dict):
        return
    assert schema, f"Empty schema at {path}"
    for keyword in ("properties", "$defs", "patternProperties"):
        for name, child in schema.get(keyword, {}).items():
            check_schema(child, f"{path}.{keyword}.{name}")
    for keyword in (
        "items",
        "additionalProperties",
        "contains",
        "not",
        "if",
        "then",
        "else",
    ):
        if keyword in schema:
            check_schema(schema[keyword], f"{path}.{keyword}")
    for keyword in ("anyOf", "oneOf", "allOf", "prefixItems"):
        for index, child in enumerate(schema.get(keyword, [])):
            check_schema(child, f"{path}.{keyword}[{index}]")


async def test_discovery_schemas_and_annotations(client: Client) -> None:
    tools = {tool.name: tool for tool in await client.list_tools()}
    assert len(tools) == 47
    for tool in tools.values():
        check_schema(tool.input_schema, f"{tool.name}.input")
        check_schema(tool.output_schema, f"{tool.name}.output")
    read = tools["view_repo"].annotations
    write = tools["create_issue"].annotations
    delete = tools["delete_repo"].annotations
    merge = tools["merge_pull_request"].annotations
    assert read is not None and write is not None
    assert delete is not None and merge is not None
    assert read.read_only_hint is True
    assert read.idempotent_hint is True
    assert write.read_only_hint is False
    assert write.idempotent_hint is False
    assert delete.destructive_hint is True
    assert merge.destructive_hint is True
    manual_merge = tools["mark_pull_request_merged"].annotations
    assert manual_merge is not None
    assert manual_merge.read_only_hint is False
    assert manual_merge.idempotent_hint is False
    assert tools["approve_pipeline"].annotations.read_only_hint is False
    assert tools["retry_pipeline"].output_schema is not None
    assert tools["pipeline_logs"].output_schema is not None
    assert "stopReason" in tools["pipeline_logs"].output_schema["properties"]


async def test_pending_attestation_tools(client: Client, fake_tg: FakeTG) -> None:
    fake_tg.configure(
        stdout='{"items":[],"warnings":["invalid queue entry left in place"]}'
    )
    listed = await client.call_tool("list_pending_attestations", {})
    assert listed.structured_content == {
        "items": [],
        "warnings": ["invalid queue entry left in place"],
    }
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        "auth",
        "attestations",
        "list",
    ]
    fake_tg.configure(stdout='{"retried":1,"pending":1,"warnings":["PDS unavailable"]}')
    retried = await client.call_tool("retry_pending_attestations", {})
    assert retried.structured_content["pending"] == 1
    assert fake_tg.requests[-1]["arguments"] == [
        "--json",
        "auth",
        "attestations",
        "retry",
    ]


async def test_saved_write_preserves_warning(client: Client, fake_tg: FakeTG) -> None:
    fake_tg.configure(
        stdout='{"rkey":"key","uri":"at://saved","warnings":["attestation pending"]}'
    )
    result = await client.call_tool(
        "create_issue", {"repo": "alice.example/repo", "title": "Saved"}
    )
    assert result.structured_content["warnings"] == ["attestation pending"]
    assert len(fake_tg.requests) == 1


@pytest.mark.parametrize(
    "tool, payload, expected_command",
    [
        (
            "approve_pipeline",
            {"pipeline": "opaque-id", "approvalRequested": True},
            "approve",
        ),
        (
            "retry_pipeline",
            {"pipeline": "new-id", "originalPipeline": "opaque-id"},
            "retry",
        ),
    ],
)
async def test_pipeline_approval_and_retry_tools(
    client: Client,
    fake_tg: FakeTG,
    tool: str,
    payload: dict[str, Any],
    expected_command: str,
) -> None:
    fake_tg.configure(stdout=json.dumps(payload))
    await client.call_tool(
        tool, {"repo": "alice.example/repo", "pipeline_id": "opaque-id"}
    )
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        "pipeline",
        expected_command,
        "--repo=alice.example/repo",
        "--",
        "opaque-id",
    ]


async def test_trigger_pipeline_accepts_sha256_without_checkout(
    client: Client, fake_tg: FakeTG
) -> None:
    commit = "a" * 64
    fake_tg.configure(stdout=json.dumps({"pipeline": "opaque-id", "commit": commit}))
    await client.call_tool(
        "trigger_pipeline", {"repo": "alice.example/repo", "revision": commit}
    )
    assert fake_tg.requests[0]["arguments"][-1] == commit


@pytest.mark.parametrize("tool", ["edit_issue", "edit_pull_request"])
async def test_expected_cid_is_passed_to_edit(
    client: Client, fake_tg: FakeTG, tool: str
) -> None:
    fake_tg.configure(stdout='{"rkey":"key","cid":"version","uri":"at://example"}')
    result = await client.call_tool(
        tool,
        {
            "repo": "alice.example/repo",
            "rkey": "key",
            "title": "New",
            "expected_cid": "version",
        },
    )
    assert "--expected-cid=version" in fake_tg.requests[0]["arguments"]
    assert result.structured_content["cid"] == "version"


async def test_trigger_pipeline_preserves_all_scheduled_pipelines(
    client: Client, fake_tg: FakeTG
) -> None:
    commit = "a" * 64
    fake_tg.configure(
        stdout=json.dumps(
            {
                "pipeline": "first",
                "pipelines": ["first", "second"],
                "message": "Started verification",
                "commit": commit,
            }
        )
    )
    result = await client.call_tool(
        "trigger_pipeline", {"repo": "alice.example/repo", "revision": commit}
    )
    assert result.structured_content["pipeline"] == "first"
    assert result.structured_content["pipelines"] == ["first", "second"]
    assert result.structured_content["message"] == "Started verification"


async def test_record_manual_merge_updates_status(
    client: Client, fake_tg: FakeTG
) -> None:
    fake_tg.configure(
        stdout=json.dumps(
            {
                "rkey": "key",
                "state": "merged",
                "uri": "at://example",
                "cid": "cid",
                "warnings": ["attestation pending"],
            }
        )
    )
    result = await client.call_tool(
        "mark_pull_request_merged",
        {
            "repo": "alice.example/repo",
            "rkey": "key",
            "expected_cid": "expected",
        },
    )
    assert not result.is_error
    assert result.structured_content == {
        "rkey": "key",
        "state": "merged",
        "uri": "at://example",
        "cid": "cid",
        "warnings": ["attestation pending"],
    }
    assert fake_tg.requests[0]["arguments"] == [
        "--json",
        "pr",
        "mark-merged",
        "--repo",
        "alice.example/repo",
        "--expected-cid=expected",
        "--",
        "key",
    ]
