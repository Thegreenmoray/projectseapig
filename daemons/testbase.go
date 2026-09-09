package daemons

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/Justi/projectseapig/runners"
)

type TestExecutor interface {
	Start() error
	RunTests(names []string) ([]runners.TestResult, error)
	Stop() error
}

type DaemonBase struct {
	Conn       net.Conn
	Proc       ProcessRunner // Replaces raw *exec.Cmd for safe mocking
	Langtype   string
	Socketpath string
}

func (p *DaemonBase) Stop() error {
	var firstErr error

	// 1. Close connection if present
	if p.Conn != nil {
		if err := p.Conn.Close(); err != nil {
			firstErr = fmt.Errorf("unable to close socket connection: %w", err)
		}
	}

	// 2. Kill daemon subprocess if present
	if p.Proc != nil {
		if err := p.Proc.Kill(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("cannot kill daemon command: %w", err)
		}
		_ = p.Proc.Wait() // Prevent zombie processes
	}

	// 3. Cleanup socket file
	if p.Socketpath != "" {
		_ = os.Remove(p.Socketpath)
	}

	return firstErr
}

func (p *DaemonBase) RunTests(testNames []string) ([]runners.TestResult, error) {
	if p.Conn == nil {
		return nil, fmt.Errorf("cannot run tests: socket connection is nil")
	}

	// Send batch over socket
	if err := json.NewEncoder(p.Conn).Encode(testNames); err != nil {
		return nil, fmt.Errorf("unable to send tests due to: %w", err)
	}

	// Read response batch
	var results []runners.TestResult
	if err := json.NewDecoder(p.Conn).Decode(&results); err != nil {
		return nil, fmt.Errorf("unable to decode test results due to: %w", err)
	}

	return results, nil
}

//we know we need a unix socket in order keep the deamon up,
// and since we know the socket will be on the same machine.
