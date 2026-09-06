package daemons

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"time"
)

type JsDaemon struct {
	DaemonBase
	Timeout     time.Duration //until a batch of tests are timed out
	DaemonPath  string        //where the deamon is located
	ProjectRoot string        //tests are located
	NodePath    string
	IsTS        bool
}

func (t *JsDaemon) StartDaemon() error {
	//we will need to startup a socket for the deamon to listen on
	args := []string{"tsx", t.DaemonPath, "--socket", t.Socketpath}
	if t.IsTS {
		args = append(args, "--ts")
	}
	cmd := exec.Command("npx", args...)

	t.DaemonBase.Cmdkill = cmd
	//returns the output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("Cannot establish pipe connection to TS")
	}
	if eee := cmd.Start(); eee != nil { //forgot to add this lol
		return fmt.Errorf("Cannot startup TS")
	}
	//Maybe add a time out to prevent this from hanging later
	//but man, I really need to brush up on my io and cmd knowledge

	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() { //is a while loop, still getting used to that.
		if scanner.Text() == "READY" {
			break
		}
	}
	//even if the ts server sets up properly the os may not catch on to that the first time. this is here to catch those cases
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("unix", t.Socketpath)
		if err != nil && i != 9 {
			time.Sleep(50 * time.Millisecond)
		} else {
			if err != nil {
				return fmt.Errorf("Cannot dial Server: %w", err)
			}

			t.DaemonBase.Conn = conn
			break //do we introduce a break here? we've got the connection at this point.
		}
	}

	//test
	fmt.Println("Jest Daemon should be running")

	return nil
}
