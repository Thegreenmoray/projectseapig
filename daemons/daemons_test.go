package daemons

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Justi/projectseapig/runners"
)

type MockProcess struct{}

func (m *MockProcess) Kill() error { return nil }
func (m *MockProcess) Wait() error { return nil }

type MockExecutor struct {
	StdoutReader io.ReadCloser
	BinaryName   string
}

type RecordingSocketDialer struct {
	Network string
	Address string
}

func (d *RecordingSocketDialer) Dial(network, address string) (net.Conn, error) {
	d.Network = network
	d.Address = address
	client, peer := net.Pipe()
	go peer.Close()
	return client, nil
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
			expectedBin: "node",
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

func TestDaemonsAcceptTCPReadyHandshake(t *testing.T) {
	tests := []struct {
		name  string
		start func(CommandExecutor, SocketDialer) TestExecutor
	}{
		{
			name: "JavaScript",
			start: func(executor CommandExecutor, dialer SocketDialer) TestExecutor {
				return &JsDaemon{
					DaemonBase: DaemonBase{Socketpath: "unused.sock"},
					DaemonPath: "server.ts",
					Executor:   executor,
					Dialer:     dialer,
				}
			},
		},
		{
			name: "Python",
			start: func(executor CommandExecutor, dialer SocketDialer) TestExecutor {
				return &PythonDaemon{
					DaemonBase: DaemonBase{Socketpath: "unused.sock"},
					DaemonPath: "server.py",
					Executor:   executor,
					Dialer:     dialer,
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader, writer := io.Pipe()
			go func() {
				_, _ = writer.Write([]byte("READY TCP 127.0.0.1:43210\n"))
				_ = writer.Close()
			}()
			dialer := &RecordingSocketDialer{}
			daemon := test.start(&MockExecutor{StdoutReader: reader}, dialer)
			if err := daemon.Start(); err != nil {
				t.Fatalf("Start() failed: %v", err)
			}
			defer daemon.Stop()
			if dialer.Network != "tcp" || dialer.Address != "127.0.0.1:43210" {
				t.Fatalf("unexpected dial target: %s %s", dialer.Network, dialer.Address)
			}
		})
	}
}

func TestJavaServerBuildUsesMavenWrapper(t *testing.T) {
	projectRoot := t.TempDir()
	serverDir := t.TempDir()
	mavenWrapper := "mvnw"
	if runtime.GOOS == "windows" {
		mavenWrapper += ".cmd"
	}
	for _, path := range []string{
		filepath.Join(projectRoot, "pom.xml"),
		filepath.Join(projectRoot, mavenWrapper),
		filepath.Join(serverDir, "pom.xml"),
	} {
		if err := os.WriteFile(path, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	command, args, err := javaServerBuildCommand(projectRoot, serverDir)
	if err != nil {
		t.Fatalf("javaServerBuildCommand() failed: %v", err)
	}

	wrapperPath := filepath.Join(projectRoot, mavenWrapper)
	wantArgs := []string{"-f", filepath.Join(serverDir, "pom.xml"), "package"}
	if runtime.GOOS == "windows" {
		wantArgs = append([]string{"/c", wrapperPath}, wantArgs...)
		if command != "cmd.exe" {
			t.Fatalf("command = %q, want cmd.exe", command)
		}
	} else if command != wrapperPath {
		t.Fatalf("command = %q, want %q", command, wrapperPath)
	}
	for index, want := range wantArgs {
		if index >= len(args) || args[index] != want {
			t.Fatalf("args = %#v, want prefix %#v", args, wantArgs)
		}
	}
}

func TestJavaServerJarRebuildsWhenSourceChanges(t *testing.T) {
	serverDir := t.TempDir()
	jarPath := filepath.Join(serverDir, "build", "libs", "server.jar")
	if err := os.MkdirAll(filepath.Dir(jarPath), 0755); err != nil {
		t.Fatal(err)
	}
	jarTime := time.Now().Add(-time.Minute)
	if err := os.WriteFile(jarPath, []byte("jar"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(jarPath, jarTime, jarTime); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(serverDir, "Server.java")
	if err := os.WriteFile(sourcePath, []byte("class Server {}"), 0600); err != nil {
		t.Fatal(err)
	}
	if needsBuild, err := javaServerJarNeedsBuild(jarPath, serverDir); err != nil || !needsBuild {
		t.Fatalf("newer source should require a rebuild: needsBuild=%t err=%v", needsBuild, err)
	}

	sourceTime := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(sourcePath, sourceTime, sourceTime); err != nil {
		t.Fatal(err)
	}
	if needsBuild, err := javaServerJarNeedsBuild(jarPath, serverDir); err != nil || needsBuild {
		t.Fatalf("older source should reuse the JAR: needsBuild=%t err=%v", needsBuild, err)
	}
}

func TestJavaDaemonBuildsAndStartsWithJDKOnly(t *testing.T) {
	for _, tool := range []string{"java", "javac", "jar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is unavailable", tool)
		}
	}

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate Java server source")
	}
	repoRoot := filepath.Dir(filepath.Dir(testFile))
	serverDir := filepath.Join(t.TempDir(), "servers")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(repoRoot, "servers", "Server.java"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "Server.java"), source, 0600); err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Join(t.TempDir(), "maven-project")
	if err := os.MkdirAll(projectRoot, 0755); err != nil {
		t.Fatal(err)
	}
	jarPath := filepath.Join(serverDir, "build", "libs", "seapig-server-1.0-SNAPSHOT.jar")
	daemon := &JavaDaemon{
		DaemonBase:  DaemonBase{Socketpath: filepath.Join(t.TempDir(), "java-daemon.sock")},
		DaemonPath:  jarPath,
		ProjectRoot: projectRoot,
	}
	if err := daemon.ensureServerJar(); err != nil {
		t.Fatalf("automatic JDK-only daemon build failed: %v", err)
	}
	if err := daemon.Start(); err != nil {
		t.Fatalf("Java daemon failed to start: %v", err)
	}
	defer daemon.Stop()
	results, err := daemon.RunTests([]string{})
	if err != nil {
		t.Fatalf("Java daemon socket request failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("empty request returned unexpected results: %+v", results)
	}
	failures, err := daemon.RunTests([]string{"MissingBuildTest"})
	if err != nil {
		t.Fatalf("Java daemon failure response could not be decoded: %v", err)
	}
	if len(failures) != 1 || failures[0].Passed || !strings.Contains(failures[0].Stderr, projectRoot) {
		t.Fatalf("unexpected escaped failure response: %+v", failures)
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
