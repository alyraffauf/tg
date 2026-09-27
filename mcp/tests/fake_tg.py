"""Subprocess fixture. Never connects to Tangled."""

import json
import os
import subprocess
import sys
import time
from pathlib import Path

scenario_path = Path(os.environ["TGMCP_TEST_SCENARIO"])
scenario = json.loads(scenario_path.read_text())
child = None
if scenario["spawn_child"]:
    child = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(60)"])
request = {
    "arguments": sys.argv[1:],
    "cwd": os.getcwd(),
    "stdin": sys.stdin.read(),
    "pid": os.getpid(),
    "child_pid": child.pid if child else None,
}
with scenario_path.with_suffix(".requests").open("a") as output:
    output.write(json.dumps(request) + "\n")
sys.stdout.write(scenario["stdout"])
sys.stdout.flush()
sys.stderr.write(scenario["stderr"])
sys.stderr.flush()
if scenario["stall"]:
    time.sleep(60)
sys.exit(scenario["exit_code"])
