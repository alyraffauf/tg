"""Fixtures shared by runner and MCP tests."""

import asyncio
import json
import os
import signal
import sys
from collections.abc import AsyncIterator, Iterator
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import pytest
from fastmcp import Client

from tgmcp import server
from tgmcp.runner import TGConfig, TGRunner


@dataclass
class FakeTG:
    executable: Path
    scenario_path: Path

    def configure(
        self,
        *,
        stdout: str = '{"rkey":"key"}',
        stderr: str = "",
        exit_code: int = 0,
        stall: bool = False,
        spawn_child: bool = False,
    ) -> None:
        self.scenario_path.write_text(
            json.dumps(
                {
                    "stdout": stdout,
                    "stderr": stderr,
                    "exit_code": exit_code,
                    "stall": stall,
                    "spawn_child": spawn_child,
                }
            )
        )

    @property
    def requests(self) -> list[dict[str, Any]]:
        path = self.scenario_path.with_suffix(".requests")
        return (
            [json.loads(line) for line in path.read_text().splitlines()]
            if path.exists()
            else []
        )

    def runner(self, **settings: Any) -> TGRunner:
        return TGRunner(TGConfig(executable=str(self.executable), **settings))

    async def wait_ready(self) -> dict[str, Any]:
        async with asyncio.timeout(5):
            while not self.requests:
                await asyncio.sleep(0.01)
        return self.requests[-1]

    async def assert_stopped(self) -> None:
        async with asyncio.timeout(5):
            while any(
                process_running(request[key])
                for request in self.requests
                for key in ("pid", "child_pid")
            ):
                await asyncio.sleep(0.01)


def process_running(pid: int | None) -> bool:
    if pid is None:
        return False
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    status = Path(f"/proc/{pid}/stat")
    try:
        return status.read_text().rsplit(")", 1)[1].split()[0] != "Z"
    except FileNotFoundError, ProcessLookupError:
        return not Path("/proc").exists()


@pytest.fixture(autouse=True)
def fake_tg(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Iterator[FakeTG]:
    executable = tmp_path / "tg"
    executable.write_text(
        f"#!{sys.executable}\n" + Path(__file__).with_name("fake_tg.py").read_text()
    )
    executable.chmod(0o700)
    fake = FakeTG(executable, tmp_path / "scenario.json")
    fake.configure()
    monkeypatch.setenv("TGMCP_TEST_SCENARIO", str(fake.scenario_path))
    monkeypatch.setattr(server, "runner", fake.runner())
    yield fake
    for request in fake.requests:
        try:
            if os.name == "posix":
                os.killpg(request["pid"], signal.SIGKILL)
            elif process_running(request["pid"]):
                os.kill(request["pid"], signal.SIGTERM)
        except ProcessLookupError:
            pass


@pytest.fixture
async def client() -> AsyncIterator[Client]:
    async with Client(server.mcp) as connected:
        yield connected


@pytest.fixture
def pipeline_json() -> str:
    return json.dumps(
        {
            "id": "pipeline",
            "commit": "a" * 40,
            "trigger": {"$type": "sh.tangled.ci.trigger#push", "newSha": "a" * 40},
            "workflows": [{"id": "build", "name": "build", "status": "failed"}],
        }
    )


@pytest.fixture
def log_line() -> str:
    return (
        json.dumps(
            {
                "type": "data",
                "data": {
                    "step": 1,
                    "time": "2026-01-01T00:00:00Z",
                    "stream": "stdout",
                    "content": "hello",
                    "workflow": "build",
                },
            }
        )
        + "\n"
    )
