package daemons

import (
	"net"
	"os/exec"
)

type DaemonBase struct {
	Conn     net.Conn  //now a client since the interper will be the server go has to be the client
	Cmdkill  *exec.Cmd //This will both reference the command that runs the interper and will kill it when finished.
	Langtype string
}

type Daemon interface {
	StartDaemon() error
	RunTests(testNames []string) ([]string, error)
	StopDaemon() error
}

//we know we need a unix socket in order keep the deamon up,
// and since we know the socket will be on the same machine.
