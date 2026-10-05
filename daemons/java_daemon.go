package daemons

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
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
	serverDir := filepath.Dir(filepath.Dir(filepath.Dir(j.DaemonPath)))
	needsBuild, err := javaServerJarNeedsBuild(j.DaemonPath, serverDir)
	if err != nil {
		return err
	}
	if !needsBuild {
		return nil
	}
	if !filepath.IsAbs(j.DaemonPath) {
		return nil
	}

	command, args, err := javaServerBuildCommand(j.ProjectRoot, serverDir)
	if err != nil {
		return j.buildServerJarWithJDK(serverDir)
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
		return fmt.Errorf("failed to build Java daemon JAR: %w\n%s", waitErr, output.String())
	}
	if _, err := os.Stat(j.DaemonPath); err != nil {
		return fmt.Errorf("Gradle completed but Java daemon JAR was not created at %q", j.DaemonPath)
	}
	return nil
}

func (j *JavaDaemon) buildServerJarWithJDK(serverDir string) error {
	javac, err := exec.LookPath("javac")
	if err != nil {
		return fmt.Errorf("Java daemon JAR is missing and no Maven/Gradle build tool is available; install a JDK with javac or provide a Maven/Gradle wrapper: %w", err)
	}
	jar, err := exec.LookPath("jar")
	if err != nil {
		return fmt.Errorf("Java daemon JAR is missing and the JDK jar tool is unavailable: %w", err)
	}
	classesDir, err := os.MkdirTemp("", "seapig-java-classes-*")
	if err != nil {
		return fmt.Errorf("cannot create Java daemon build directory: %w", err)
	}
	defer os.RemoveAll(classesDir)
	if err := os.MkdirAll(filepath.Dir(j.DaemonPath), 0755); err != nil {
		return fmt.Errorf("cannot create Java daemon JAR directory: %w", err)
	}

	sourcePath := filepath.Join(serverDir, "Server.java")
	output, err := exec.Command(javac, "-d", classesDir, sourcePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to compile Java daemon with javac: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	output, err = exec.Command(jar, "--create", "--file", j.DaemonPath, "--main-class", "servers.Server", "-C", classesDir, ".").CombinedOutput()
	if err != nil {
		_ = os.Remove(j.DaemonPath)
		return fmt.Errorf("failed to package Java daemon with jar: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func javaServerJarNeedsBuild(jarPath, serverDir string) (bool, error) {
	jarInfo, err := os.Stat(jarPath)
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("cannot inspect Java daemon JAR %q: %w", jarPath, err)
	}

	for _, source := range []string{"Server.java", "build.gradle", "pom.xml"} {
		info, err := os.Stat(filepath.Join(serverDir, source))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("cannot inspect Java daemon source %q: %w", source, err)
		}
		if info.ModTime().After(jarInfo.ModTime()) {
			return true, nil
		}
	}
	return false, nil
}

func javaServerBuildCommand(projectRoot, serverDir string) (string, []string, error) {
	isWindows := os.PathSeparator == '\\'
	gradleWrapper := "gradlew"
	mavenWrapper := "mvnw"
	if isWindows {
		gradleWrapper += ".bat"
		mavenWrapper += ".cmd"
	}
	gradleWrapperPath := filepath.Join(projectRoot, gradleWrapper)
	mavenWrapperPath := filepath.Join(projectRoot, mavenWrapper)
	serverPom := filepath.Join(serverDir, "pom.xml")
	serverGradle := false
	for _, buildFile := range []string{"build.gradle", "build.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(serverDir, buildFile)); err == nil {
			serverGradle = true
			break
		}
	}
	serverMaven := false
	if _, err := os.Stat(serverPom); err == nil {
		serverMaven = true
	}
	projectPom := filepath.Join(projectRoot, "pom.xml")
	projectGradle := false
	for _, buildFile := range []string{"build.gradle", "build.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(projectRoot, buildFile)); err == nil {
			projectGradle = true
			break
		}
	}
	projectMaven := false
	if _, err := os.Stat(projectPom); err == nil {
		projectMaven = true
	}

	if projectMaven && serverMaven {
		if _, err := os.Stat(mavenWrapperPath); err == nil {
			command, args := wrapperCommand(mavenWrapperPath, []string{"-f", serverPom, "package"}, isWindows)
			return command, args, nil
		}
		if mvnPath, err := exec.LookPath("mvn"); err == nil {
			command, args := wrapperCommand(mvnPath, []string{"-f", serverPom, "package"}, isWindows)
			return command, args, nil
		}
	}

	gradleArgs := []string{"-p", serverDir, "jar"}
	if projectGradle && serverGradle {
		if _, err := os.Stat(gradleWrapperPath); err == nil {
			command, args := wrapperCommand(gradleWrapperPath, gradleArgs, isWindows)
			return command, args, nil
		}
	}
	if serverGradle {
		if gradlePath, err := exec.LookPath("gradle"); err == nil {
			command, args := wrapperCommand(gradlePath, gradleArgs, isWindows)
			return command, args, nil
		}
	}
	if serverGradle {
		if _, err := os.Stat(gradleWrapperPath); err == nil {
			command, args := wrapperCommand(gradleWrapperPath, gradleArgs, isWindows)
			return command, args, nil
		}
	}
	if serverMaven {
		if _, err := os.Stat(mavenWrapperPath); err == nil {
			command, args := wrapperCommand(mavenWrapperPath, []string{"-f", serverPom, "package"}, isWindows)
			return command, args, nil
		}
	}
	if serverMaven {
		if mvnPath, err := exec.LookPath("mvn"); err == nil {
			command, args := wrapperCommand(mvnPath, []string{"-f", serverPom, "package"}, isWindows)
			return command, args, nil
		}
	}
	return "", nil, fmt.Errorf("Java daemon JAR is missing; install Gradle or Maven, or provide gradlew/mvnw in %s", projectRoot)
}

func wrapperCommand(executable string, args []string, isWindows bool) (string, []string) {
	if isWindows {
		return "cmd.exe", append([]string{"/c", executable}, args...)
	}
	return executable, args
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

	//fmt.Println("JUnit Daemon running successfully!")
	return nil
}
