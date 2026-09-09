package daemons

import (
	"bufio"
	"fmt"
	"time"
)

type JsDaemon struct {
	DaemonBase
	Timeout     time.Duration
	DaemonPath  string
	ProjectRoot string
	NodePath    string
	IsTS        bool

	Executor CommandExecutor
	Dialer   SocketDialer
}

func (t *JsDaemon) getExecutor() CommandExecutor {
	if t.Executor == nil {
		return &RealCommandExecutor{}
	}
	return t.Executor
}

func (t *JsDaemon) getDialer() SocketDialer {
	if t.Dialer == nil {
		return &RealSocketDialer{}
	}
	return t.Dialer
}

func (t *JsDaemon) Start() error {
	args := []string{"tsx", t.DaemonPath, "--socket", t.Socketpath}
	if t.IsTS {
		args = append(args, "--ts")
	}

	proc, stdoutPipe, err := t.getExecutor().StartCommand("npx", args...)
	if err != nil {
		return fmt.Errorf("Cannot startup TS/JS daemon: %w", err)
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
		return fmt.Errorf("Error reading TS/JS daemon stdout: %w", err)
	}
	if !readyReceived {
		return fmt.Errorf("TS/JS daemon process exited before sending READY")
	}

	dialer := t.getDialer()
	for i := 0; i < 10; i++ {
		conn, err := dialer.Dial("unix", t.Socketpath)
		if err == nil {
			t.DaemonBase.Conn = conn
			break // Connection acquired! Stop retry loop immediately.
		}

		if i == 9 {
			_ = proc.Kill()
			return fmt.Errorf("Cannot dial TS/JS daemon Server: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("Jest Daemon running successfully!")
	return nil
}
