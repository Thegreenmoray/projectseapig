package servers;

import com.google.gson.Gson;
import com.google.gson.annotations.SerializedName;
import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.IOException;
import java.nio.channels.Channels;
import java.nio.channels.ServerSocketChannel;
import java.nio.channels.SocketChannel;
import java.net.StandardProtocolFamily;
import java.net.UnixDomainSocketAddress;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

class TestResult {
    @SerializedName("test_name")
    private final String testName;
    @SerializedName("passed")
    private final boolean passed;
    @SerializedName("time_taken")
    private final long timeTaken;
    @SerializedName("stdout")
    private final String stdout;
    @SerializedName("stderr")
    private final String stderr;

    TestResult(String testName, boolean passed, long timeTaken, String stdout, String stderr) {
        this.testName = testName;
        this.passed = passed;
        this.timeTaken = timeTaken;
        this.stdout = stdout;
        this.stderr = stderr;
    }
}

public class Server {
    private static final Gson GSON = new Gson();

    private static TestResult runTest(String testName, Path projectRoot) {
        List<String> command = new ArrayList<>();
        Path gradleBuild = projectRoot.resolve("build.gradle");
        Path gradleKotlinBuild = projectRoot.resolve("build.gradle.kts");
        Path mavenBuild = projectRoot.resolve("pom.xml");
        boolean isWindows = System.getProperty("os.name").toLowerCase(Locale.ROOT).contains("win");

        if (Files.exists(gradleBuild) || Files.exists(gradleKotlinBuild)) {
            Path wrapper = projectRoot.resolve(isWindows ? "gradlew.bat" : "gradlew");
            if (Files.exists(wrapper)) {
                if (isWindows) {
                    command.add("cmd.exe");
                    command.add("/c");
                }
                command.add(wrapper.toString());
            } else {
                command.add("gradle");
            }
            command.add("test");
            command.add("--tests");
            command.add(testName);
            command.add("--console=plain");
        } else if (Files.exists(mavenBuild)) {
            Path wrapper = projectRoot.resolve(isWindows ? "mvnw.cmd" : "mvnw");
            if (Files.exists(wrapper)) {
                if (isWindows) {
                    command.add("cmd.exe");
                    command.add("/c");
                }
                command.add(wrapper.toString());
            } else {
                command.add("mvn");
            }
            command.add("-Dtest=" + testName);
            command.add("-Dstyle.color=never");
            command.add("test");
        } else {
            return new TestResult(testName, false, 0, "", "No Gradle or Maven build file found in " + projectRoot);
        }

        long start = System.nanoTime();
        try {
            Process process = new ProcessBuilder(command)
                .directory(projectRoot.toFile())
                .redirectErrorStream(true)
                .start();
            String output = new String(process.getInputStream().readAllBytes(), StandardCharsets.UTF_8);
            int exitCode = process.waitFor();
            long duration = System.nanoTime() - start;
            return new TestResult(testName, exitCode == 0, duration, output, exitCode == 0 ? "" : output);
        } catch (IOException error) {
            return new TestResult(testName, false, System.nanoTime() - start, "", error.toString());
        } catch (InterruptedException error) {
            Thread.currentThread().interrupt();
            return new TestResult(testName, false, System.nanoTime() - start, "", "Test execution interrupted");
        }
    }

    private static List<TestResult> runTests(String[] testNames, Path projectRoot) {
        List<TestResult> results = new ArrayList<>();
        for (String testName : testNames) {
            results.add(runTest(testName, projectRoot));
        }
        return results;
    }

    private static String argument(String[] args, String name) {
        for (int i = 0; i + 1 < args.length; i++) {
            if (name.equals(args[i])) {
                return args[i + 1];
            }
        }
        return null;
    }

    public static void main(String[] args) throws IOException {
        String socketPathValue = argument(args, "--socket");
        String projectRootValue = argument(args, "--project-root");
        if (socketPathValue == null || projectRootValue == null) {
            System.err.println("Required arguments: --socket and --project-root");
            System.exit(1);
        }

        Path socketPath = Paths.get(socketPathValue);
        Path projectRoot = Paths.get(projectRootValue).toAbsolutePath().normalize();
        Path socketParent = socketPath.toAbsolutePath().getParent();
        if (socketParent != null) {
            Files.createDirectories(socketParent);
        }
        Files.deleteIfExists(socketPath);

        try (ServerSocketChannel server = ServerSocketChannel.open(StandardProtocolFamily.UNIX)) {
            server.bind(UnixDomainSocketAddress.of(socketPath));
            System.out.println("READY");
            System.out.flush();

            while (true) {
                try (SocketChannel client = server.accept();
                     BufferedReader reader = new BufferedReader(Channels.newReader(client, StandardCharsets.UTF_8));
                     BufferedWriter writer = new BufferedWriter(Channels.newWriter(client, StandardCharsets.UTF_8))) {
                    String line;
                    while ((line = reader.readLine()) != null) {
                        try {
                            String[] testNames = GSON.fromJson(line, String[].class);
                            if (testNames == null) {
                                throw new IllegalArgumentException("request must be a JSON array of test selectors");
                            }
                            writer.write(GSON.toJson(runTests(testNames, projectRoot)));
                        } catch (RuntimeException error) {
                            writer.write(GSON.toJson(List.of(
                                new TestResult("<daemon>", false, 0, "", error.toString())
                            )));
                        }
                        writer.newLine();
                        writer.flush();
                    }
                } catch (IOException error) {
                    System.err.println("Client session closed or reset: " + error.getMessage());
                }
            }
        } finally {
            Files.deleteIfExists(socketPath);
        }
    }
}