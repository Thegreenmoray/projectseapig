package daemons

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type JavaDaemon struct {
	DaemonBase
	Timeout     time.Duration
	DaemonPath  string
	TestPath    string
	ProjectRoot string
	IsKotlin    bool

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

func (j *JavaDaemon) ensureServerJar() error {
	if _, err := os.Stat(j.DaemonPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot inspect Java daemon JAR %q: %w", j.DaemonPath, err)
	}
	if !filepath.IsAbs(j.DaemonPath) {
		return nil
	}

	serverDir := filepath.Dir(filepath.Dir(filepath.Dir(j.DaemonPath)))
	wrapperName := "gradlew"
	if os.PathSeparator == '\\' {
		wrapperName = "gradlew.bat"
	}
	wrapperPath := filepath.Join(j.ProjectRoot, wrapperName)
	command := "gradle"
	args := []string{"-p", serverDir, "jar"}
	if _, err := os.Stat(wrapperPath); err == nil {
		if os.PathSeparator == '\\' {
			command = "cmd.exe"
			args = append([]string{"/c", wrapperPath}, args...)
		} else {
			command = wrapperPath
		}
	}

	proc, stdout, err := j.getExecutor().StartCommand(command, args...)
	if err != nil {
		return fmt.Errorf("Java daemon JAR is missing and Gradle could not be started; run gradle -p %q jar: %w", serverDir, err)
	}
	var output strings.Builder
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		output.WriteString(scanner.Text())
		output.WriteByte('\n')
	}
	scanErr := scanner.Err()
	waitErr := proc.Wait()
	if scanErr != nil {
		return fmt.Errorf("error reading Gradle output while building Java daemon JAR: %w", scanErr)
	}
	if waitErr != nil {
		return fmt.Errorf("failed to build Java daemon JAR with Gradle: %w\n%s", waitErr, output.String())
	}
	if _, err := os.Stat(j.DaemonPath); err != nil {
		return fmt.Errorf("Gradle completed but Java daemon JAR was not created at %q", j.DaemonPath)
	}
	return nil
}

func (j *JavaDaemon) Start() error {
	if err := j.ensureServerJar(); err != nil {
		return err
	}

	args := []string{
		"-jar", j.DaemonPath,
		"--socket", j.Socketpath,
		"--project-root", j.ProjectRoot,
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
		_ = proc.Kill()
		_ = proc.Wait()
		return fmt.Errorf("error reading Java daemon stdout: %w", err)
	}
	if !readyReceived {
		_ = proc.Kill()
		_ = proc.Wait()
		return fmt.Errorf("java daemon exited before sending READY")
	}
	j.Proc = proc
	go drainProcessOutput(scanner)

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
			_ = proc.Wait()
			j.Proc = nil
			return fmt.Errorf("cannot dial Java daemon server after retries: %w", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Println("JUnit Daemon running successfully!")
	return nil
}
