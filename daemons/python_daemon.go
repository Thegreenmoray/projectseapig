package daemons

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	python := "python"
	if p.ProjectRoot != "" {
		for _, candidate := range []string{
			filepath.Join(p.ProjectRoot, ".venv", "Scripts", "python.exe"),
			filepath.Join(p.ProjectRoot, "venv", "Scripts", "python.exe"),
			filepath.Join(p.ProjectRoot, ".venv", "bin", "python"),
			filepath.Join(p.ProjectRoot, "venv", "bin", "python"),
		} {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				python = candidate
				break
			}
		}
	}
	args := []string{p.DaemonPath, "--socket", p.Socketpath}
	if p.Timeout > 0 {
		args = append(args, "--timeout-ns", strconv.FormatInt(p.Timeout.Nanoseconds(), 10))
	}
	if p.ProjectRoot != "" {
		args = append(args, "--project-root", p.ProjectRoot)
	}
	proc, stdoutPipe, err := p.getExecutor().StartCommand(python, args...)
	if err != nil {
		return fmt.Errorf("Cannot startup Python daemon: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	readyReceived := false
	socketNetwork := "unix"
	socketAddress := p.Socketpath
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
		return fmt.Errorf("Error reading Python daemon stdout: %w", err)
	}
	if !readyReceived {
		_ = proc.Kill()
		_ = proc.Wait()
		return fmt.Errorf("Python daemon process exited before sending READY")
	}
	p.Proc = proc
	go drainProcessOutput(scanner)

	dialer := p.getDialer()
	for i := 0; i < 10; i++ {
		conn, err := dialer.Dial(socketNetwork, socketAddress)
		if err == nil {
			p.Conn = conn
			break // Connection acquired! Stop retry loop immediately.
		}

		if i == 9 {
			_ = proc.Kill()
			_ = proc.Wait()
			p.Proc = nil
			return fmt.Errorf("Cannot dial Python daemon Server: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("Pytest Daemon running successfully!")
	return nil
}
