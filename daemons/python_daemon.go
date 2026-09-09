package daemons

import (
	"bufio"
	"fmt"
	"time"
)

type PythonDaemon struct {
	DaemonBase
	Timeout     time.Duration
	DaemonPath  string
	ProjectRoot string

	Executor CommandExecutor
	Dialer   SocketDialer
}

func (p *PythonDaemon) getExecutor() CommandExecutor {
	if p.Executor == nil {
		return &RealCommandExecutor{}
	}
	return p.Executor
}

func (p *PythonDaemon) getDialer() SocketDialer {
	if p.Dialer == nil {
		return &RealSocketDialer{}
	}
	return p.Dialer
}

func (p *PythonDaemon) Start() error {
	proc, stdoutPipe, err := p.getExecutor().StartCommand("python", p.DaemonPath, "--socket", p.Socketpath)
	if err != nil {
		return fmt.Errorf("Cannot startup Python daemon: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	readyReceived := false
	for scanner.Scan() {
		if scanner.Text() == "READY" {
			readyReceived = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading Python daemon stdout: %w", err)
	}
	if !readyReceived {
		return fmt.Errorf("Python daemon process exited before sending READY")
	}

	dialer := p.getDialer()
	for i := 0; i < 10; i++ {
		conn, err := dialer.Dial("unix", p.Socketpath)
		if err == nil {
			p.Conn = conn
			break // Connection acquired! Stop retry loop immediately.
		}

		if i == 9 {
			_ = proc.Kill()
			return fmt.Errorf("Cannot dial Python daemon Server: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("Pytest Daemon running successfully!")
	return nil
}
