package servers;

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
import java.util.concurrent.TimeUnit;
import javax.xml.parsers.DocumentBuilderFactory;
import javax.xml.parsers.ParserConfigurationException;
import org.w3c.dom.Element;
import org.w3c.dom.Node;
import org.w3c.dom.NodeList;
import org.xml.sax.SAXException;

class TestResult {
    final String testName;
    final boolean passed;
    final long timeTaken;
    final String stdout;
    final String stderr;
    final boolean timedOut;

    TestResult(String testName, boolean passed, long timeTaken, String stdout, String stderr) {
		this(testName, passed, timeTaken, stdout, stderr, false);
	}

	TestResult(String testName, boolean passed, long timeTaken, String stdout, String stderr, boolean timedOut) {
        this.testName = testName;
        this.passed = passed;
        this.timeTaken = timeTaken;
        this.stdout = stdout;
        this.stderr = stderr;
        this.timedOut = timedOut;
    }

    String toJson() {
        return "{\"test_name\":" + Server.quoteJsonString(testName)
            + ",\"passed\":" + passed
            + ",\"time_taken\":" + timeTaken
            + ",\"timed_out\":" + timedOut
            + ",\"stdout\":" + Server.quoteJsonString(stdout)
            + ",\"stderr\":" + Server.quoteJsonString(stderr) + "}";
    }
}

public class Server {
    private static List<TestResult> runTest(String testName, Path projectRoot, long timeoutMs) {
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
                if (isWindows) {
                    command.add("cmd.exe");
                    command.add("/c");
                }
                command.add("gradle");
            }
            command.add("test");
            command.add("--tests");
            command.add(testName);
            command.add("--console=plain");
            command.add("--rerun-tasks");
        } else if (Files.exists(mavenBuild)) {
            Path wrapper = projectRoot.resolve(isWindows ? "mvnw.cmd" : "mvnw");
            if (Files.exists(wrapper)) {
                if (isWindows) {
                    command.add("cmd.exe");
                    command.add("/c");
                }
                command.add(wrapper.toString());
            } else {
                if (isWindows) {
                    command.add("cmd.exe");
                    command.add("/c");
                }
                command.add("mvn");
            }
            command.add("-Dtest=" + testName);
            command.add("-Dstyle.color=never");
            command.add("test");
        } else {
            return List.of(new TestResult(testName, false, 0, "", "No Gradle or Maven build file found in " + projectRoot));
        }

        long start = System.nanoTime();
        Path outputFile = null;
        try {
            outputFile = Files.createTempFile(projectRoot, ".seapig-test-", ".log");
            Process process = new ProcessBuilder(command)
                .directory(projectRoot.toFile())
                .redirectErrorStream(true)
                .redirectOutput(outputFile.toFile())
                .start();
            if (!process.waitFor(timeoutMs, TimeUnit.MILLISECONDS)) {
                process.descendants().forEach(ProcessHandle::destroyForcibly);
                process.destroyForcibly();
                process.waitFor();
                String output = Files.readString(outputFile, StandardCharsets.UTF_8);
                return List.of(new TestResult(
                    testName,
                    false,
                    System.nanoTime() - start,
                    output,
                    "test timed out after " + timeoutMs + " ms",
                    true
                ));
            }
            String output = Files.readString(outputFile, StandardCharsets.UTF_8);
            int exitCode = process.exitValue();
            long duration = System.nanoTime() - start;
            List<TestResult> junitResults = readJUnitResults(projectRoot, testName);
            if (!junitResults.isEmpty()) {
                return junitResults;
            }
            return List.of(new TestResult(testName, exitCode == 0, duration, output, exitCode == 0 ? "" : output));
        } catch (IOException error) {
            return List.of(new TestResult(testName, false, System.nanoTime() - start, "", error.toString()));
        } catch (InterruptedException error) {
            Thread.currentThread().interrupt();
            return List.of(new TestResult(testName, false, System.nanoTime() - start, "", "Test execution interrupted"));
        } finally {
            if (outputFile != null) {
                try {
                    Files.deleteIfExists(outputFile);
                } catch (IOException ignored) {
                    // Keep cleanup failure from masking the test result.
                }
            }
        }
    }

    private static List<TestResult> runTests(String[] testNames, Path projectRoot, long timeoutMs) {
        List<TestResult> results = new ArrayList<>();
        for (String testName : testNames) {
            results.addAll(runTest(testName, projectRoot, timeoutMs));
        }
        return results;
    }

    private static List<TestResult> readJUnitResults(Path projectRoot, String testName) throws IOException {
        List<TestResult> results = new ArrayList<>();
        Path[] reportDirectories = {
            projectRoot.resolve("build/test-results/test"),
            projectRoot.resolve("target/surefire-reports")
        };

        for (Path reportDirectory : reportDirectories) {
            if (!Files.isDirectory(reportDirectory)) {
                continue;
            }

            try (var reportFiles = Files.list(reportDirectory)) {
                for (Path reportFile : reportFiles.toList()) {
                    String filename = reportFile.getFileName().toString();
                    if (!filename.startsWith("TEST-") || !filename.endsWith(".xml")
                            || !matchesTestClass(filename.substring(5, filename.length() - 4), testName)) {
                        continue;
                    }
                    results.addAll(parseJUnitReport(reportFile, testName));
                }
            }
        }
        return results;
    }

    private static boolean matchesTestClass(String reportClass, String testName) {
        return reportClass.equals(testName)
            || (!testName.contains(".") && reportClass.endsWith("." + testName));
    }

    private static List<TestResult> parseJUnitReport(Path reportFile, String testName) throws IOException {
        try {
            DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();
            factory.setFeature("http://apache.org/xml/features/disallow-doctype-decl", true);
            factory.setFeature("http://xml.org/sax/features/external-general-entities", false);
            factory.setFeature("http://xml.org/sax/features/external-parameter-entities", false);
            factory.setAttribute(javax.xml.XMLConstants.ACCESS_EXTERNAL_DTD, "");
            factory.setAttribute(javax.xml.XMLConstants.ACCESS_EXTERNAL_SCHEMA, "");

            var document = factory.newDocumentBuilder().parse(reportFile.toFile());
            NodeList testCases = document.getElementsByTagName("testcase");
            List<TestResult> results = new ArrayList<>(testCases.getLength());
            for (int i = 0; i < testCases.getLength(); i++) {
                Element testCase = (Element) testCases.item(i);
                String className = testCase.getAttribute("classname");
                if (!className.isEmpty() && !matchesTestClass(className, testName)) {
                    continue;
                }

                String failure = childText(testCase, "failure");
                String error = childText(testCase, "error");
                String name = testCase.getAttribute("name");
                String resultName = (className.isEmpty() ? testName : className) + "::" + name;
                String seconds = testCase.getAttribute("time");
                long duration = 0;
                try {
                    duration = (long) (Double.parseDouble(seconds) * 1_000_000_000L);
                } catch (NumberFormatException ignored) {
                    // Some test frameworks omit or leave the duration blank.
                }

                String stderr = !failure.isEmpty() ? failure : error;
                results.add(new TestResult(
                    resultName,
                    failure.isEmpty() && error.isEmpty(),
                    duration,
                    childText(testCase, "system-out"),
                    stderr,
                    isTimeoutMessage(stderr)
                ));
            }
            return results;
        } catch (ParserConfigurationException | SAXException error) {
            throw new IOException("cannot parse JUnit XML report " + reportFile + ": " + error.getMessage(), error);
        }
    }

    private static String childText(Element parent, String tagName) {
        NodeList children = parent.getChildNodes();
        for (int i = 0; i < children.getLength(); i++) {
            Node child = children.item(i);
            if (child instanceof Element element && tagName.equals(element.getTagName())) {
                return element.getTextContent();
            }
        }
        return "";
    }

    private static String argument(String[] args, String name) {
        for (int i = 0; i + 1 < args.length; i++) {
            if (name.equals(args[i])) {
                return args[i + 1];
            }
        }
        return null;
    }

    private static boolean isTimeoutMessage(String message) {
        String normalized = message.toLowerCase(Locale.ROOT);
        return normalized.contains("timed out") || normalized.contains("timeout");
    }

    private static String[] parseTestNames(String json) {
        return new JsonStringArrayParser(json).parse();
    }

    private static String serializeResults(List<TestResult> results) {
        StringBuilder json = new StringBuilder("[");
        for (int i = 0; i < results.size(); i++) {
            if (i > 0) {
                json.append(',');
            }
            json.append(results.get(i).toJson());
        }
        return json.append(']').toString();
    }

    static String quoteJsonString(String value) {
        StringBuilder json = new StringBuilder(value.length() + 2).append('"');
        char[] hex = "0123456789abcdef".toCharArray();
        for (int i = 0; i < value.length(); i++) {
            char character = value.charAt(i);
            switch (character) {
                case '"' -> json.append("\\\"");
                case '\\' -> json.append("\\\\");
                case '\b' -> json.append("\\b");
                case '\f' -> json.append("\\f");
                case '\n' -> json.append("\\n");
                case '\r' -> json.append("\\r");
                case '\t' -> json.append("\\t");
                default -> {
                    if (character < 0x20) {
                        json.append("\\u00")
                            .append(hex[(character >> 4) & 0xf])
                            .append(hex[character & 0xf]);
                    } else {
                        json.append(character);
                    }
                }
            }
        }
        return json.append('"').toString();
    }

    private static final class JsonStringArrayParser {
        private final String input;
        private int position;

        JsonStringArrayParser(String input) {
            this.input = input;
        }

        String[] parse() {
            List<String> values = new ArrayList<>();
            skipWhitespace();
            expect('[');
            skipWhitespace();
            if (consume(']')) {
                finish();
                return new String[0];
            }

            while (true) {
                skipWhitespace();
                values.add(parseString());
                skipWhitespace();
                if (consume(']')) {
                    finish();
                    return values.toArray(new String[0]);
                }
                expect(',');
            }
        }

        private String parseString() {
            expect('"');
            StringBuilder value = new StringBuilder();
            while (position < input.length()) {
                char character = input.charAt(position++);
                if (character == '"') {
                    return value.toString();
                }
                if (character < 0x20) {
                    throw new IllegalArgumentException("unescaped control character in JSON string");
                }
                if (character != '\\') {
                    value.append(character);
                    continue;
                }
                if (position >= input.length()) {
                    throw new IllegalArgumentException("incomplete escape in JSON string");
                }
                char escape = input.charAt(position++);
                switch (escape) {
                    case '"', '\\', '/' -> value.append(escape);
                    case 'b' -> value.append('\b');
                    case 'f' -> value.append('\f');
                    case 'n' -> value.append('\n');
                    case 'r' -> value.append('\r');
                    case 't' -> value.append('\t');
                    case 'u' -> value.append(parseUnicodeEscape());
                    default -> throw new IllegalArgumentException("invalid JSON string escape");
                }
            }
            throw new IllegalArgumentException("unterminated JSON string");
        }

        private char parseUnicodeEscape() {
            if (position + 4 > input.length()) {
                throw new IllegalArgumentException("incomplete unicode escape in JSON string");
            }
            int value = 0;
            for (int i = 0; i < 4; i++) {
                int digit = Character.digit(input.charAt(position++), 16);
                if (digit < 0) {
                    throw new IllegalArgumentException("invalid unicode escape in JSON string");
                }
                value = (value << 4) | digit;
            }
            return (char) value;
        }

        private void finish() {
            skipWhitespace();
            if (position != input.length()) {
                throw new IllegalArgumentException("unexpected content after JSON array");
            }
        }

        private void skipWhitespace() {
            while (position < input.length()) {
                char character = input.charAt(position);
                if (character != ' ' && character != '\t' && character != '\r' && character != '\n') {
                    return;
                }
                position++;
            }
        }

        private boolean consume(char expected) {
            if (position < input.length() && input.charAt(position) == expected) {
                position++;
                return true;
            }
            return false;
        }

        private void expect(char expected) {
            if (!consume(expected)) {
                throw new IllegalArgumentException("expected '" + expected + "' in JSON array");
            }
        }
    }

    public static void main(String[] args) throws IOException {
        String socketPathValue = argument(args, "--socket");
        String projectRootValue = argument(args, "--project-root");
        String timeoutValue = argument(args, "--timeout-ms");
        if (socketPathValue == null || projectRootValue == null) {
            System.err.println("Required arguments: --socket and --project-root");
            System.exit(1);
        }

        Path socketPath = Paths.get(socketPathValue);
        Path projectRoot = Paths.get(projectRootValue).toAbsolutePath().normalize();
        long timeoutMs = 600_000;
        try {
            long configuredTimeout = Long.parseLong(timeoutValue);
            if (configuredTimeout > 0) {
                timeoutMs = configuredTimeout;
            }
        } catch (NumberFormatException ignored) {
            // Use the default test timeout when no valid value is supplied.
        }
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
                            String[] testNames = parseTestNames(line);
                            writer.write(serializeResults(runTests(testNames, projectRoot, timeoutMs)));
                        } catch (RuntimeException error) {
                            writer.write(serializeResults(List.of(
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