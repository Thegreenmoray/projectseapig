import argparse
import json
import os
import socket
import subprocess
import sys
import pytest


class SeapigCollector:
    """Pytest plugin hook to capture test outcomes in memory."""

    def __init__(self):
        self.results = []
        self.current_result = None

    def pytest_runtest_logstart(self, nodeid, location):
        self.current_result = {
            "test_name": nodeid,
            "passed": True,
            "time_taken": 0,
            "stdout": "",
            "stderr": "",
        }
        self.results.append(self.current_result)

    def pytest_runtest_logreport(self, report):
        if report.when not in ("setup", "call", "teardown"):
            return

        result = self.current_result
        if result is None or result["test_name"] != report.nodeid:
            result = {
                "test_name": report.nodeid,
                "passed": True,
                "time_taken": 0,
                "stdout": "",
                "stderr": "",
            }
            self.results.append(result)

        result["time_taken"] += int(report.duration * 1e9)
        if report.failed:
            result["passed"] = False
            result["stderr"] = str(report.longrepr) if report.longrepr else "Test failed"


def run_test_process(test_name, project_root):
    process = subprocess.run(
        [sys.executable, os.path.abspath(__file__), "--run-selector", test_name],
        cwd=project_root,
        capture_output=True,
        text=True,
    )
    marker = "SEAPIG_RESULT:"
    for line in reversed(process.stdout.splitlines()):
        if line.startswith(marker):
            return json.loads(line[len(marker):])

    output = process.stderr or process.stdout or f"pytest exited with code {process.returncode}"
    return [{
        "test_name": test_name,
        "passed": False,
        "time_taken": 0,
        "stdout": process.stdout,
        "stderr": output,
    }]

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--socket", help="Path to Unix socket")
    parser.add_argument("--project-root", default=os.getcwd())
    parser.add_argument("--run-selector")
    args = parser.parse_args()

    if args.run_selector is not None:
        collector = SeapigCollector()
        pytest.main(["-q", args.run_selector], plugins=[collector])
        print("SEAPIG_RESULT:" + json.dumps(collector.results), flush=True)
        return
    if args.socket is None:
        parser.error("--socket is required")

    socket_path = args.socket
    os.chdir(args.project_root)
    os.makedirs(os.path.dirname(os.path.abspath(socket_path)), exist_ok=True)
    use_tcp = os.name == "nt"
    if not use_tcp and os.path.exists(socket_path):
        os.remove(socket_path)

    try:
        family = socket.AF_INET if use_tcp else socket.AF_UNIX
        with socket.socket(family, socket.SOCK_STREAM) as server:
            if use_tcp:
                server.bind(("127.0.0.1", 0))
            else:
                server.bind(socket_path)
            server.listen(1)
            if use_tcp:
                print(f"READY TCP 127.0.0.1:{server.getsockname()[1]}", flush=True)
            else:
                print("READY", flush=True)

            while True:
                conn, _ = server.accept()
                with conn:
                    message_buffer = b""
                    while True:
                        data = conn.recv(4096)
                        if not data:
                            break
                        message_buffer += data
                        while b"\n" in message_buffer:
                            line, message_buffer = message_buffer.split(b"\n", 1)
                            if not line.strip():
                                continue
                            try:
                                test_names = json.loads(line.decode("utf-8"))
                                if not isinstance(test_names, list) or not all(isinstance(name, str) for name in test_names):
                                    raise ValueError("request must be a JSON array of test selectors")
                                response = []
                                for test_name in test_names:
                                    response.extend(run_test_process(test_name, args.project_root))
                            except Exception as error:
                                response = [{
                                    "test_name": "<daemon>",
                                    "passed": False,
                                    "time_taken": 0,
                                    "stdout": "",
                                    "stderr": str(error),
                                }]
                            conn.sendall(json.dumps(response).encode("utf-8") + b"\n")
    finally:
        if not use_tcp and os.path.exists(socket_path):
            os.remove(socket_path)


if __name__ == "__main__":
    main()          


