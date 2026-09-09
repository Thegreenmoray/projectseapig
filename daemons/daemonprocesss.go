package daemons

import (
	"io"
	"net"
	"os"
	"os/exec"
)

type CommandExecutor interface {
	StartCommand(name string, args ...string) (ProcessRunner, io.ReadCloser, error)
}

type ProcessRunner interface {
	Kill() error
	Wait() error
}

type RealCommandExecutor struct{}

type realProcess struct {
	cmd *exec.Cmd
}

func (r *realProcess) Kill() error { return r.cmd.Process.Kill() }
func (r *realProcess) Wait() error { return r.cmd.Wait() }

func (r *RealCommandExecutor) StartCommand(name string, args ...string) (ProcessRunner, io.ReadCloser, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return &realProcess{cmd: cmd}, stdout, nil
}

type SocketDialer interface {
	Dial(network, address string) (net.Conn, error)
}

type RealSocketDialer struct{}

func (d *RealSocketDialer) Dial(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}
