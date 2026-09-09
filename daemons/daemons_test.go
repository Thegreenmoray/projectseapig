package daemons

import (
	"encoding/json"
	"io"
	"net"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Justi/projectseapig/runners"
)

type MockProcess struct{}

func (m *MockProcess) Kill() error { return nil }
func (m *MockProcess) Wait() error { return nil }

type MockExecutor struct {
	StdoutReader io.ReadCloser
	BinaryName   string
}

func (m *MockExecutor) StartCommand(name string, args ...string) (ProcessRunner, io.ReadCloser, error) {
	m.BinaryName = name
	return &MockProcess{}, m.StdoutReader, nil
}

func TestAllDaemons_FullLifecycle(t *testing.T) {
	tests := []struct {
		name         string
		expectedBin  string
		createDaemon func(socketPath string, exec CommandExecutor) TestExecutor
	}{
		{
			name:        "JavaDaemon",
			expectedBin: "java",
			createDaemon: func(sp string, exec CommandExecutor) TestExecutor {
				d := &JavaDaemon{
					DaemonPath: "runner.jar",
					IsKotlin:   false,
					Executor:   exec,
				}
				d.Socketpath = sp
				d.Langtype = "java"
				return d
			},
		},
		{
			name:        "JsDaemon",
			expectedBin: "npx",
			createDaemon: func(sp string, exec CommandExecutor) TestExecutor {
				d := &JsDaemon{
					DaemonPath: "runner.ts",
					IsTS:       true,
					Executor:   exec,
				}
				d.Socketpath = sp
				d.Langtype = "javascript"
				return d
			},
		},
		{
			name:        "PythonDaemon",
			expectedBin: "python",
			createDaemon: func(sp string, exec CommandExecutor) TestExecutor {
				d := &PythonDaemon{
					DaemonPath: "runner.py",
					Executor:   exec,
				}
				d.Socketpath = sp
				d.Langtype = "python"
				return d
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Setup mock unix socket server that acts like the daemon
			socketPath := filepath.Join(t.TempDir(), "daemon.sock")
			listener, err := net.Listen("unix", socketPath)
			if err != nil {
				t.Fatalf("Failed to create socket listener: %v", err)
			}
			defer listener.Close()

			// Socket server goroutine to mock daemon responding to RunTests
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()

				// Read requested test batch from client
				var testNames []string
				if err := json.NewDecoder(conn).Decode(&testNames); err != nil {
					return
				}

				// Write back fake results
				var mockResults []runners.TestResult
				for _, name := range testNames {
					mockResults = append(mockResults, runners.TestResult{
						Testname: name,
						Passed:   true,
						Stdout:   "OK",
					})
				}
				_ = json.NewEncoder(conn).Encode(mockResults)
			}()

			// 2. Setup stdout handshake pipe ("READY\n")
			pr, pw := io.Pipe()
			go func() {
				_, _ = pw.Write([]byte("READY\n"))
				_ = pw.Close()
			}()

			mockExec := &MockExecutor{StdoutReader: pr}
			daemon := tt.createDaemon(socketPath, mockExec)

			// 3. Test Start()
			if err := daemon.Start(); err != nil {
				t.Fatalf("Start() failed: %v", err)
			}
			if mockExec.BinaryName != tt.expectedBin {
				t.Errorf("Expected binary %s, got %s", tt.expectedBin, mockExec.BinaryName)
			}

			// 4. Test RunTests() over socket
			results, err := daemon.RunTests([]string{"TestOne", "TestTwo"})
			if err != nil {
				t.Fatalf("RunTests() failed: %v", err)
			}
			if len(results) != 2 || !results[0].Passed {
				t.Errorf("Unexpected RunTests output: %+v", results)
			}

			// 5. Test Stop()
			if err := daemon.Stop(); err != nil {
				t.Fatalf("Stop() failed: %v", err)
			}
		})
	}
}

func TestRealCommandExecutor_StartAndWait(t *testing.T) {
	exec := &RealCommandExecutor{}

	cmdName := "echo"
	cmdArgs := []string{"READY"}
	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		cmdArgs = []string{"/c", "echo READY"}
	}

	proc, stdout, err := exec.StartCommand(cmdName, cmdArgs...)
	if err != nil {
		t.Fatalf("StartCommand failed: %v", err)
	}
	if stdout == nil {
		t.Fatal("Expected non-nil stdout reader")
	}

	if err := proc.Wait(); err != nil {
		t.Errorf("Wait failed: %v", err)
	}
}

func TestRealCommandExecutor_Kill(t *testing.T) {
	exec := &RealCommandExecutor{}

	cmdName := "sleep"
	cmdArgs := []string{"5"}
	if runtime.GOOS == "windows" {
		cmdName = "ping"
		cmdArgs = []string{"127.0.0.1", "-n", "6"} // Windows equivalent of sleep 5
	}

	proc, _, err := exec.StartCommand(cmdName, cmdArgs...)
	if err != nil {
		t.Fatalf("StartCommand failed: %v", err)
	}

	if err := proc.Kill(); err != nil {
		t.Errorf("Kill failed: %v", err)
	}
	_ = proc.Wait()
}
