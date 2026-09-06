package daemons

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"time"
)

type PythonDaemon struct {
	DaemonBase
	Timeout     time.Duration //until a batch of tests are timed out
	DaemonPath  string        //where the deamon is located
	ProjectRoot string
}

// Ill do python on my own since its less convoluted and not tied to multiple sub languages
//unlike java which has kotlin and js which has ts.

func (p *PythonDaemon) StartDaemon() error {
	//we will need to startup a socket for the deamon to listen on
	cmd := exec.Command("python", p.DaemonPath, "--socket", p.Socketpath)

	p.Cmdkill = cmd
	//returns the output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Cannot establish pipe connection to Python")
	}
	if eee := cmd.Start(); eee != nil { //forgot to add this lol
		return fmt.Errorf("Cannot startup Python")
	}
	//Maybe add a time out to prevent this from hanging later
	//but man, I really need to brush up on my io and cmd knowledge

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() { //is a while loop, still getting used to that.
		if scanner.Text() == "READY" {
			break
		}
	}
	//even if the python server sets up properly the os may not catch on to that the first time. this is here to catch those cases
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("unix", p.Socketpath)
		if err != nil && i != 9 {
			time.Sleep(50 * time.Millisecond)
		} else {
			if err != nil {
				return fmt.Errorf("Cannot dial Server: %w", err)
			}

			p.Conn = conn
			break //do we introduce a break here? we've got the connection at this point.
		}
	}

	//test
	fmt.Println("Pytest Daemon should be running")

	return nil
}
