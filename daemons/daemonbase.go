package daemons

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/Justi/projectseapig/runners"
)

type DaemonBase struct {
	Conn       net.Conn  //now a client since the interper will be the server go has to be the client
	Cmdkill    *exec.Cmd //This will both reference the command that runs the interper and will kill it when finished.
	Langtype   string
	Socketpath string // Path to the Unix socket for communication with the Python daemon
}

type Daemon interface {
	StartDaemon() error
	RunTests(testNames []string) ([]runners.TestResult, error)
	StopDaemon() error
}

func (p *DaemonBase) StopDaemon() error {
	//kill the deamon when we're done with it
	if p.Conn != nil {
		return fmt.Errorf("Connection is invaild")
	}

	if eer := p.Conn.Close(); eer != nil {
		return fmt.Errorf("Unable to close connection due to: %w", eer)
	}
	if p.Cmdkill != nil && p.Cmdkill.Process != nil {
		return fmt.Errorf("Command is invaild")
	}

	if er := p.Cmdkill.Process.Kill(); er != nil {
		return fmt.Errorf("Cannot kill command due to: %s", er)
	}
	_ = p.Cmdkill.Wait() //stops zombies

	if p.Socketpath != "" {
		_ = os.Remove(p.Socketpath)
	}

	return nil
}

func (p *DaemonBase) RunTests(testNames []string) ([]runners.TestResult, error) {
	wrapped := json.NewEncoder(p.Conn)                //since these tests are being sent by an array of strings we have to wrap it in json
	if err := wrapped.Encode(testNames); err != nil { //converting to bytes and sending to server
		return nil, fmt.Errorf("Unable to send tests due to: %w", err)
	}
	var marinesnow []runners.TestResult //just a biology reference dont worry too much about it
	decode := json.NewDecoder(p.Conn)
	if err := decode.Decode(&marinesnow); err != nil { //decode will halt the thread (gorountie in this case), basic pbr/pbv stuff here.
		return nil, fmt.Errorf("Unable to decode message due to: %w", err)
	}
	return marinesnow, nil
}

//we know we need a unix socket in order keep the deamon up,
// and since we know the socket will be on the same machine.
