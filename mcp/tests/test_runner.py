"""Command output and process lifecycle tests."""

import asyncio
from pathlib import Path

import pytest

from tests.conftest import FakeTG
from tgmcp.models import UpdatedRecordResult
from tgmcp.runner import TGConfig, TGError, TGRunner


async def test_json_result(fake_tg: FakeTG) -> None:
    result = await fake_tg.runner().run_json(
        ["issue", "edit", "--", "key"], UpdatedRecordResult
    )
    assert result.rkey == "key"
    assert fake_tg.requests[0]["arguments"] == ["--json", "issue", "edit", "--", "key"]


@pytest.mark.parametrize("stdout", ["not json", "{}", '{"rkey":123}'])
async def test_invalid_output(fake_tg: FakeTG, stdout: str) -> None:
    fake_tg.configure(stdout=stdout)
    with pytest.raises(TGError, match="did not match UpdatedRecordResult"):
        await fake_tg.runner().run_json([], UpdatedRecordResult)


@pytest.mark.parametrize(
    "stderr, message",
    [("permission denied", "permission denied"), ("", "exited with status 1")],
)
async def test_command_failure(fake_tg: FakeTG, stderr: str, message: str) -> None:
    fake_tg.configure(stderr=stderr, exit_code=1)
    with pytest.raises(TGError, match=message):
        await fake_tg.runner().run_json([], UpdatedRecordResult)
    assert len(fake_tg.requests) == 1


async def test_missing_executable(tmp_path: Path) -> None:
    runner = TGRunner(TGConfig(executable=str(tmp_path / "missing")))
    with pytest.raises(TGError, match="Could not start tg"):
        await runner.run_json([], UpdatedRecordResult)


async def test_configuration_cwd_and_stdin(fake_tg: FakeTG, tmp_path: Path) -> None:
    runner = fake_tg.runner(account="alice.example", config_path="/tmp/config.toml")
    await runner.run_json(
        ["string", "create"],
        UpdatedRecordResult,
        working_directory=str(tmp_path),
        input_data="héllo\n".encode(),
    )
    request = fake_tg.requests[0]
    assert request["arguments"] == [
        "--json",
        "--account",
        "alice.example",
        "--config",
        "/tmp/config.toml",
        "string",
        "create",
    ]
    assert request["cwd"] == str(tmp_path)
    assert request["stdin"] == "héllo\n"


async def test_relative_cwd_rejected(fake_tg: FakeTG) -> None:
    with pytest.raises(TGError, match="absolute path"):
        await fake_tg.runner().run_json(
            [], UpdatedRecordResult, working_directory="relative"
        )
    assert not fake_tg.requests


async def test_timeout_stops_process_tree(fake_tg: FakeTG) -> None:
    fake_tg.configure(stall=True, spawn_child=True)
    with pytest.raises(TGError, match="timed out"):
        await asyncio.wait_for(
            fake_tg.runner(timeout_seconds=1).run_json([], UpdatedRecordResult), 5
        )
    assert fake_tg.requests
    await fake_tg.assert_stopped()


async def test_cancellation_stops_process_tree(fake_tg: FakeTG) -> None:
    fake_tg.configure(stall=True, spawn_child=True)
    task = asyncio.create_task(fake_tg.runner().run_json([], UpdatedRecordResult))
    try:
        await fake_tg.wait_ready()
        task.cancel()
        with pytest.raises(asyncio.CancelledError):
            await asyncio.wait_for(task, 5)
        await fake_tg.assert_stopped()
    finally:
        task.cancel()
        await asyncio.gather(task, return_exceptions=True)
