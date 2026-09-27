"""Run tg commands without a shell."""

import asyncio
import math
import os
import signal
from dataclasses import dataclass

from fastmcp.exceptions import ToolError
from pydantic import BaseModel, ValidationError

from tgmcp.models import PipelineLogEvent, PipelineLogsResult


class TGError(ToolError):
    """A tg command could not produce a valid result."""


@dataclass(frozen=True)
class TGConfig:
    executable: str = "tg"
    account: str | None = None
    config_path: str | None = None
    timeout_seconds: float = 30.0

    def __post_init__(self) -> None:
        if not self.executable:
            raise ValueError("TGMCP_EXECUTABLE must not be empty")
        if not math.isfinite(self.timeout_seconds) or self.timeout_seconds <= 0:
            raise ValueError("TGMCP_TIMEOUT_SECONDS must be finite and positive")

    @classmethod
    def from_env(cls) -> TGConfig:
        return cls(
            executable=os.environ.get("TGMCP_EXECUTABLE", "tg"),
            account=os.environ.get("TGMCP_ACCOUNT") or None,
            config_path=os.environ.get("TGMCP_CONFIG") or None,
            timeout_seconds=float(os.environ.get("TGMCP_TIMEOUT_SECONDS", "30")),
        )


class TGRunner:
    def __init__(self, config: TGConfig) -> None:
        self.config = config

    async def run_json[Output: BaseModel](
        self,
        arguments: list[str],
        output_model: type[Output],
        *,
        expected_failure: tuple[int, str] | None = None,
        working_directory: str | None = None,
        input_data: bytes | None = None,
    ) -> Output:
        stdout = await self._run(
            arguments,
            expected_failure=expected_failure,
            working_directory=working_directory,
            input_data=input_data,
        )
        try:
            return output_model.model_validate_json(stdout)
        except ValidationError as error:
            raise TGError(
                f"tg output did not match {output_model.__name__}: {error}. "
                "Check that tg is up to date. If this was a write, inspect current "
                "state before retrying. The operation may have succeeded."
            ) from error

    async def _start(
        self,
        arguments: list[str],
        *,
        working_directory: str | None = None,
        input_data: bytes | None = None,
    ) -> asyncio.subprocess.Process:
        if working_directory is not None and not os.path.isabs(working_directory):
            raise TGError(
                "working_directory must be an absolute path on the server host, "
                "such as /home/user/src/repo"
            )
        command = [self.config.executable, "--json"]
        if self.config.account is not None:
            command.extend(["--account", self.config.account])
        if self.config.config_path is not None:
            command.extend(["--config", self.config.config_path])
        command.extend(arguments)

        try:
            return await asyncio.create_subprocess_exec(
                *command,
                stdin=asyncio.subprocess.PIPE
                if input_data is not None
                else asyncio.subprocess.DEVNULL,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=working_directory,
                start_new_session=os.name == "posix",
                limit=1 << 20,
            )
        except FileNotFoundError as error:
            if working_directory is not None and not os.path.isdir(working_directory):
                message = (
                    f"Working directory {working_directory!r} does not exist. "
                    "Use an existing checkout or create one with clone_repo."
                )
            else:
                message = (
                    f"Executable {self.config.executable!r} was not found. Install tg "
                    "on the server host or set TGMCP_EXECUTABLE to its absolute path."
                )
            raise TGError(f"Could not start tg: {message}") from error
        except OSError as error:
            raise TGError(f"Could not start tg: {error}") from error

    async def _stop(self, process: asyncio.subprocess.Process) -> None:
        try:
            if os.name == "posix":
                os.killpg(process.pid, signal.SIGKILL)
            else:
                process.kill()
        except ProcessLookupError:
            pass
        await process.communicate()

    async def _run(
        self,
        arguments: list[str],
        *,
        expected_failure: tuple[int, str] | None = None,
        working_directory: str | None = None,
        input_data: bytes | None = None,
    ) -> bytes:
        process = await self._start(
            arguments, working_directory=working_directory, input_data=input_data
        )
        try:
            async with asyncio.timeout(self.config.timeout_seconds):
                stdout, stderr = await process.communicate(input_data)
        except (TimeoutError, asyncio.CancelledError) as error:
            await self._stop(process)
            if isinstance(error, asyncio.CancelledError):
                raise
            raise TGError(
                f"tg timed out after {self.config.timeout_seconds:g} seconds. "
                "Inspect current state before repeating a write. For slow reads, "
                "narrow the request or increase TGMCP_TIMEOUT_SECONDS."
            ) from error

        if process.returncode != 0:
            message = stderr.decode("utf-8", errors="replace").strip()
            if (process.returncode, message) != expected_failure:
                raise TGError(
                    self._command_error(arguments, process.returncode, message)
                )

        return stdout

    def _command_error(
        self, arguments: list[str], status: int | None, message: str
    ) -> str:
        context = " ".join(arguments[:2])
        detail = message or f"tg exited with status {status} without an error message"
        if "not logged in" in message.lower():
            detail += (
                ". Check the account with auth_status or list_accounts. Ask the "
                "user to run tg auth login <handle> on the server host."
            )
        elif 'repo "' in message and "not found for handle" in message:
            detail += ". Use search_repos or list_repos to find the handle/repo target."
        return f"tg {context}: {detail}" if context else detail

    async def collect_logs(
        self,
        arguments: list[str],
        *,
        max_events: int,
        max_bytes: int,
        timeout_seconds: float,
    ) -> PipelineLogsResult:
        process = await self._start(arguments)
        assert process.stdout is not None and process.stderr is not None
        result = PipelineLogsResult(stop_reason="complete")

        async def read_events() -> None:
            assert process.stdout is not None
            byte_count = 0
            while True:
                try:
                    line = await process.stdout.readline()
                except ValueError:
                    result.stop_reason = "byte_limit"
                    return
                if not line:
                    return
                byte_count += len(line)
                if byte_count > max_bytes:
                    result.stop_reason = "byte_limit"
                    return
                try:
                    result.events.append(PipelineLogEvent.model_validate_json(line))
                except ValidationError as error:
                    result.stop_reason = "error"
                    result.error = f"Invalid pipeline log event: {error}"
                    return
                if len(result.events) >= max_events:
                    result.stop_reason = "event_limit"
                    return

        async def read_errors() -> bytes:
            assert process.stderr is not None
            captured = bytearray()
            while chunk := await process.stderr.read(65536):
                captured.extend(chunk[: max(0, 65536 - len(captured))])
            return bytes(captured)

        events_task = asyncio.create_task(read_events())
        errors_task = asyncio.create_task(read_errors())
        try:
            async with asyncio.timeout(
                min(timeout_seconds, self.config.timeout_seconds)
            ):
                await events_task
                if result.stop_reason == "complete":
                    await process.wait()
                    stderr = await errors_task
                    if process.returncode != 0:
                        result.stop_reason = "error"
                        result.error = self._command_error(
                            arguments,
                            process.returncode,
                            stderr.decode("utf-8", errors="replace").strip(),
                        )
        except TimeoutError:
            result.stop_reason = "timeout"
        finally:
            events_task.cancel()
            errors_task.cancel()
            await asyncio.gather(events_task, errors_task, return_exceptions=True)
            await self._stop(process)
        return result
