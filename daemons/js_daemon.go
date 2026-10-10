package daemons

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
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
	tsxCLI := filepath.Join(filepath.Dir(t.DaemonPath), "node_modules", "tsx", "dist", "cli.mjs")
	args := []string{tsxCLI, t.DaemonPath, "--socket", t.Socketpath}
	if t.Timeout > 0 {
		args = append(args, "--timeout-ms", strconv.FormatInt(t.Timeout.Milliseconds(), 10))
	}
	if t.NodePath != "" {
		args = append(args, "--project-root", t.NodePath)
	}
	if t.IsTS {
		args = append(args, "--ts")
	}

	proc, stdoutPipe, err := t.getExecutor().StartCommand("node", args...)
	if err != nil {
		return fmt.Errorf("Cannot startup TS/JS daemon: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	readyReceived := false
	socketNetwork := "unix"
	socketAddress := t.Socketpath
	for scanner.Scan() {
		line := scanner.Text()
		if line == "READY" {
			readyReceived = true
			break
		}
		if strings.HasPrefix(line, "READY TCP ") {
			socketNetwork = "tcp"
			socketAddress = strings.TrimPrefix(line, "READY TCP ")
			readyReceived = socketAddress != ""
			break
		}
	}
	if err := scanner.Err(); err != nil {
		_ = proc.Kill()
		_ = proc.Wait()
		return fmt.Errorf("Error reading TS/JS daemon stdout: %w", err)
	}
	if !readyReceived {
		_ = proc.Kill()
		_ = proc.Wait()
		return fmt.Errorf("TS/JS daemon process exited before sending READY")
	}
	t.Proc = proc
	go drainProcessOutput(scanner)

	dialer := t.getDialer()
	for i := 0; i < 10; i++ {
		conn, err := dialer.Dial(socketNetwork, socketAddress)
		if err == nil {
			t.DaemonBase.Conn = conn
			break // Connection acquired! Stop retry loop immediately.
		}

		if i == 9 {
			_ = proc.Kill()
			_ = proc.Wait()
			t.Proc = nil
			return fmt.Errorf("Cannot dial TS/JS daemon Server: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}
