"""Bounded log collection and stream cleanup tests."""

import asyncio

import pytest

from tests.conftest import FakeTG


@pytest.mark.parametrize("ending", ["\n", ""])
async def test_complete_stream(fake_tg: FakeTG, log_line: str, ending: str) -> None:
    fake_tg.configure(stdout=log_line.rstrip("\n") + ending)
    result = await fake_tg.runner().collect_logs(
        [], max_events=10, max_bytes=4096, timeout_seconds=2
    )
    assert result.stop_reason == "complete"
    assert len(result.events) == 1
    assert result.events[0].data is not None
    assert result.events[0].data.content == "hello"
    assert result.error is None


@pytest.mark.parametrize("limit", ["events", "bytes"])
async def test_output_limits_keep_prior_events(
    fake_tg: FakeTG, log_line: str, limit: str
) -> None:
    fake_tg.configure(stdout=log_line * 3, stall=True, spawn_child=True)
    result = await asyncio.wait_for(
        fake_tg.runner().collect_logs(
            [],
            max_events=1 if limit == "events" else 10,
            max_bytes=len(log_line.encode()) if limit == "bytes" else 4096,
            timeout_seconds=2,
        ),
        5,
    )
    assert result.stop_reason == ("event_limit" if limit == "events" else "byte_limit")
    assert len(result.events) == 1
    await fake_tg.assert_stopped()


@pytest.mark.parametrize("server_timeout, collection_timeout", [(1, 10), (10, 1)])
async def test_timeout_keeps_events(
    fake_tg: FakeTG, log_line: str, server_timeout: int, collection_timeout: int
) -> None:
    fake_tg.configure(stdout=log_line, stall=True, spawn_child=True)
    result = await asyncio.wait_for(
        fake_tg.runner(timeout_seconds=server_timeout).collect_logs(
            [], max_events=10, max_bytes=4096, timeout_seconds=collection_timeout
        ),
        5,
    )
    assert result.stop_reason == "timeout"
    assert len(result.events) == 1
    await fake_tg.assert_stopped()


@pytest.mark.parametrize("failure", ["invalid_json", "invalid_model", "exit"])
async def test_stream_errors_keep_events(
    fake_tg: FakeTG, log_line: str, failure: str
) -> None:
    suffix = {"invalid_json": "not json\n", "invalid_model": "{}\n", "exit": ""}[
        failure
    ]
    fake_tg.configure(
        stdout=log_line + suffix,
        stderr="stream failed" if failure == "exit" else "",
        exit_code=1 if failure == "exit" else 0,
    )
    result = await fake_tg.runner().collect_logs(
        [], max_events=10, max_bytes=4096, timeout_seconds=2
    )
    assert result.stop_reason == "error"
    assert len(result.events) == 1
    assert result.error


async def test_oversized_line(fake_tg: FakeTG) -> None:
    fake_tg.configure(stdout="x" * (2 << 20) + "\n", stall=True, spawn_child=True)
    result = await asyncio.wait_for(
        fake_tg.runner().collect_logs(
            [], max_events=10, max_bytes=1048576, timeout_seconds=2
        ),
        5,
    )
    assert result.stop_reason == "byte_limit"
    assert not result.events
    await fake_tg.assert_stopped()


async def test_cancellation_stops_stream_tree(fake_tg: FakeTG, log_line: str) -> None:
    fake_tg.configure(stdout=log_line, stall=True, spawn_child=True)
    task = asyncio.create_task(
        fake_tg.runner().collect_logs(
            [], max_events=10, max_bytes=4096, timeout_seconds=10
        )
    )
    try:
        await fake_tg.wait_ready()
        task.cancel()
        with pytest.raises(asyncio.CancelledError):
            await asyncio.wait_for(task, 5)
        await fake_tg.assert_stopped()
    finally:
        task.cancel()
        await asyncio.gather(task, return_exceptions=True)


async def test_log_step_outcome_survives_collection(fake_tg: FakeTG) -> None:
    fake_tg.configure(
        stdout='{"type":"control","control":{"kind":"user","step":0,"time":"2026-10-28T00:00:00Z","status":"end","outcome":"failed","content":"build","workflow":"ci.yml"}}\n'
    )
    result = await fake_tg.runner().collect_logs(
        [], max_events=10, max_bytes=4096, timeout_seconds=2
    )
    assert result.events[0].control.outcome == "failed"
