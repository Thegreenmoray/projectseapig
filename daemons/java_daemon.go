package daemons

import (
	"bufio"
	"fmt"
	"net"
	"os"
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

func (j *JavaDaemon) StartDaemon() error { // 1. Determine compiled class directory based on IsKotlin
	classDir := "build/classes/java/test"
	if j.IsKotlin {
		classDir = "build/classes/kotlin/test"
	}

	// Optional: Pass the test class path and language mode to the daemon as args
	args := []string{
		"-jar", j.DaemonPath,
		"--socket", j.Socketpath,
		"--test-classes", classDir,
	}

	if j.IsKotlin {
		args = append(args, "--mode", "kotlin")
	}

	//^this is just to ensure kotlin compatability

	cmd := exec.Command("java", args...)
	j.Cmdkill = cmd

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cannot establish stdout pipe to Java daemon: %w", err)
	}

	// Pipe stderr to standard OS stderr for easy crash debugging
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot startup Java daemon: %w", err)
	}

	// Wait for READY handshake from Java
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		if scanner.Text() == "READY" {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading Java daemon stdout: %w", err)
	}

	// Retry dial loop for OS socket propagation
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("unix", j.Socketpath)
		if err == nil {
			j.Conn = conn
			break // Connection acquired! Stop looping.
		}

		if i == 9 {
			return fmt.Errorf("cannot dial Java daemon server after retries: %w", err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("JUnit Daemon running successfully!")
	return nil
}
