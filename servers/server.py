import argparse
import json
import os
import socket
import sys
import pytest


class SeapigCollector:
    """Pytest plugin hook to capture test outcomes in memory."""

    def __init__(self):
        self.results = []

    def pytest_runtest_logreport(self, report):
        if report.when == "call":
            longrepr = ""
            if report.failed and report.longrepr:
                longrepr = str(report.longrepr)

            self.results.append(
                {
                    "test_name": report.nodeid,
                    "passed": report.passed,
                    "time_taken": int(report.duration * 1e9),  # convert seconds to nanoseconds
                    "stderr": longrepr,
                }
            )

def main():
   parser = argparse.ArgumentParser()
   parser.add_argument("--socket", required=True, help="Path to Unix socket")
   args = parser.parse_args()

   socket_path = args.socket


   SOCKET_PATH = "/tmp/my_unix_socket.s"

# 1. Clean up old socket files if they exist from a dirty shutdown
   if os.path.exists(SOCKET_PATH):
    os.remove(SOCKET_PATH)

   with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as server:
     #connecting to the port
    server.bind(SOCKET_PATH)

    #now listening for go client
    server.listen(1)

    while True:
        try:
            
            print("READY") #dont bother waiting for the loop to finish
            sys.stdout.flush()
            conn,_=server.accept()
            with conn:
             
             while True:
                 data = conn.recv(4096)
                 if not data:
                    break
                 request=json.loads(data.decode('utf-8')) #8-bit Unicode Transformation Format, in this case we are converting from binary to actual test
                 test_names=request.get("tests",[])
                 collector = SeapigCollector()
                 pytest.main(["-q"] + test_names, plugins=[collector])

                # Send structured results back to Go
                 response_bytes = json.dumps(collector.results).encode("utf-8") + b"\n"
                 conn.sendall(response_bytes)
        finally:
              server.close()
              if os.path.exists(socket_path):
               os.remove(socket_path)


if __name__ == "__main__":
    main()          


