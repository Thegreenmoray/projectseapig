package daemons

import (
	"bufio"
	"fmt"
	"time"
)

type JavaDaemon struct {
	DaemonBase
	Timeout    time.Duration
	DaemonPath string
	TestPath   string
	IsKotlin   bool

	Executor CommandExecutor
	Dialer   SocketDialer
}

func (j *JavaDaemon) getExecutor() CommandExecutor {
	if j.Executor == nil {
		return &RealCommandExecutor{}
	}
	return j.Executor
}

func (j *JavaDaemon) getDialer() SocketDialer {
	if j.Dialer == nil {
		return &RealSocketDialer{}
	}
	return j.Dialer
}

func (j *JavaDaemon) Start() error {
	classDir := "build/classes/java/test"
	if j.IsKotlin {
		classDir = "build/classes/kotlin/test"
	}

	args := []string{
		"-jar", j.DaemonPath,
		"--socket", j.Socketpath,
		"--test-classes", classDir,
	}

	if j.IsKotlin {
		args = append(args, "--mode", "kotlin")
	}

	proc, stdoutPipe, err := j.getExecutor().StartCommand("java", args...)
	if err != nil {
		return fmt.Errorf("cannot startup Java daemon: %w", err)
	}

	// Read READY handshake
	scanner := bufio.NewScanner(stdoutPipe)
	readyReceived := false
	for scanner.Scan() {
		if scanner.Text() == "READY" {
			readyReceived = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading Java daemon stdout: %w", err)
	}
	if !readyReceived {
		return fmt.Errorf("java daemon exited before sending READY")
	}

	// Dial socket
	dialer := j.getDialer()
	for i := 0; i < 10; i++ {
		conn, err := dialer.Dial("unix", j.Socketpath)
		if err == nil {
			j.Conn = conn
			break
		}
		if i == 9 {
			_ = proc.Kill()
			return fmt.Errorf("cannot dial Java daemon server after retries: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("JUnit Daemon running successfully!")
	return nil
}
