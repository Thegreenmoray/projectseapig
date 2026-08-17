package daemons

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type PythonDaemon struct {
	daemonBase  DaemonBase
	Socketpath  string        // Path to the Unix socket for communication with the Python daemon
	Timeout     time.Duration //until a batch of tests are timed out
	DeamonPath  string        //where the deamon is located
	ProjectRoot string
}

// Ill do python on my own since its less convoluted and not tied to multiple sub languages
//unlike java which has kotlin and js which has ts.

func (p *PythonDaemon) StartDaemon() error {
	//we will need to startup a socket for the deamon to listen on
	cmd := exec.Command("python", p.DeamonPath, "--socket", p.Socketpath)

	p.daemonBase.Cmdkill = cmd
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

			p.daemonBase.Conn = conn
			break //do we introduce a break here? we've got the connection at this point.
		}
	}

	//test
	fmt.Println("Pytest Daemon should be running")

	return nil
}

func (p *PythonDaemon) StopDaemon() error {
	//kill the deamon when we're done with it
	if p.daemonBase.Conn != nil {
		return fmt.Errorf("Connection is invaild")
	}

	if eer := p.daemonBase.Conn.Close(); eer != nil {
		return fmt.Errorf("Unable to close connection due to: %w", eer)
	}
	if p.daemonBase.Cmdkill != nil && p.daemonBase.Cmdkill.Process != nil {
		return fmt.Errorf("Command is invaild")
	}

	if er := p.daemonBase.Cmdkill.Process.Kill(); er != nil {
		return fmt.Errorf("Cannot kill command due to: %s", er)
	}
	_ = p.daemonBase.Cmdkill.Wait() //stops zombies

	if p.Socketpath != "" {
		_ = os.Remove(p.Socketpath)
	}

	return nil
}

// Not compelte yet, later change this when we finish start and stop daemon.
func (p *PythonDaemon) RunTests(testNames []string) ([]runners.TestResult, error) {
	wrapped := json.NewEncoder(p.daemonBase.Conn)     //since these tests are being sent by an array of strings we have to wrap it in json
	if err := wrapped.Encode(testNames); err != nil { //converting to bytes and sending to server
		return nil, fmt.Errorf("Unable to send tests due to: %w", err)
	}
	var marinesnow []runners.TestResult //just a biology reference dont worry too much about it
	decode := json.NewDecoder(p.daemonBase.Conn)
	if err := decode.Decode(&marinesnow); err != nil { //decode will halt the thread (gorountie in this case), basic pbr/pbv stuff here.
		return nil, fmt.Errorf("Unable to decode message due to: %w", err)
	}
	return marinesnow, nil
}
