package daemons

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"time"
)

type JavaDaemon struct {
	DaemonBase
	Socketpath string        // Path to the Unix socket for communication with the Java daemon
	Timeout    time.Duration //until a batch of tests are timed out
	DaemonPath string        //where the deamon is located
	TestPath   string        //where the test folder is located
	IsKotlin   bool          // Flag to indicate if the project is a Kotlin project
}

func (j *JavaDaemon) StartDaemon() error {
	//we will need to startup a socket for the deamon to listen on
	cmd := exec.Command("python", j.DaemonPath, "--socket", j.Socketpath)

	j.Cmdkill = cmd
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
	//even if the java server sets up properly the os may not catch on to that the first time. this is here to catch those cases
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("unix", j.Socketpath)
		if err != nil && i != 9 {
			time.Sleep(50 * time.Millisecond)
		} else {
			if err != nil {
				return fmt.Errorf("Cannot dial Server: %w", err)
			}

			p.daemonBase.Conn = conn
			break //do we introduce a break here? we've got the connection at this point.
		}
	}

	//test
	fmt.Println("Pytest Daemon should be running")

	return nil
}
