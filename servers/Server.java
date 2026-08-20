package servers;
import com.google.gson.Gson;
import java.io.IOException;
import java.net.StandardProtocolFamily;
import java.net.UnixDomainSocketAddress;
import java.nio.ByteBuffer;
import java.nio.channels.ServerSocketChannel;
import java.nio.channels.SocketChannel;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Arrays;
import java.nio.charset.StandardCharsets;
import java.util.List;
import org.junit.platform.launcher.TestExecutionListener;
import org.junit.platform.launcher.TestIdentifier;
import org.junit.platform.launcher.TestExecutionResult;
import org.junit.platform.launcher.Launcher;
import org.junit.platform.launcher.LauncherDiscoveryRequest;
import org.junit.platform.launcher.core.LauncherFactory;
import org.junit.platform.launcher.core.LauncherDiscoveryRequestBuilder;
import org.junit.platform.engine.discovery.DiscoverySelectors;

class SeaPigCollector implements TestExecutionListener{
    private final List<TestResult> results = new ArrayList<>();

    @Override
    public void executionFinished(TestIdentifier testIdentifier, TestExecutionResult testExecutionResult) {
        // Ignore containers (like test classes) and only record actual test methods
        if (testIdentifier.isTest()) {
            boolean passed = testExecutionResult.getStatus() == TestExecutionResult.Status.SUCCESSFUL;
            
            String failureOutput = "";
            if (!passed && testExecutionResult.getThrowable().isPresent()) {
                Throwable t = testExecutionResult.getThrowable().get();
                failureOutput = t.getMessage() != null ? t.getMessage() : t.toString();
            }

            // Create your result object (matching Go's expected JSON format)
            TestResult result = new TestResult(
                testIdentifier.getUniqueId(), // or testIdentifier.getDisplayName()
                passed,
                0, // duration in nanoseconds (can track start/end times if needed)
                failureOutput
            );

            results.add(result);
        }
    }

    public List<TestResult> getResults() {
        return results;}
    }

 class TestResult {
    private String name;
    private boolean passed;
    private long duration;
    private String output;

    public TestResult(String name, boolean passed, long duration, String output) {
        this.name = name;
        this.passed = passed;
        this.duration = duration;
        this.output = output;
    }
}

public class Server {
    public static List<TestResult> runTests(List<String> testClassNames) {
        // 1. Create a discovery request for the incoming test classes from Go
        LauncherDiscoveryRequestBuilder requestBuilder = LauncherDiscoveryRequestBuilder.request();
        
        for (String className : testClassNames) {
            requestBuilder.selectors(DiscoverySelectors.selectClass(className));
        }
        
        LauncherDiscoveryRequest request = requestBuilder.build();

        // 2. Instantiate the JUnit Platform Launcher
        Launcher launcher = LauncherFactory.create();

        // 3. Instantiate your collector listener
        SeaPigCollector collector = new SeaPigCollector();

        // 4. Register the listener and execute the tests in-memory
        launcher.registerTestExecutionListeners(collector);
        launcher.execute(request);

        // 5. Retrieve accumulated test results
        return collector.getResults();
    }


    
    public static void main(String[] args) throws IOException {
    String socketPathStr = null;

    // Scan args for the --socket flag
    for (int i = 0; i < args.length; i++) {
        if ("--socket".equals(args[i]) && i + 1 < args.length) {
            socketPathStr = args[i + 1];
            break;
        }
    }

    if (socketPathStr == null) {
        System.err.println("Error: Missing required --socket argument");
        System.exit(1);
    }

    // Declare Path from the parsed CLI argument string
    Path socketPath = Path.of(socketPathStr);

    // Clean up any existing socket file from previous runs
    Files.deleteIfExists(socketPath);

    // Instantiate the address layout
    UnixDomainSocketAddress address = UnixDomainSocketAddress.of(socketPath);
    Gson gson = new Gson();

    // Create and bind the server channel
    try (ServerSocketChannel serverChannel = ServerSocketChannel.open(StandardProtocolFamily.UNIX)) {
        serverChannel.bind(address);
        System.out.println("READY");

        ByteBuffer buffer = ByteBuffer.allocate(4096);
        StringBuilder messageBuffer = new StringBuilder();

        while (true) {
            // Wait for a client connection
            try (SocketChannel clientChannel = serverChannel.accept()) {
                while (clientChannel.read(buffer) > 0) {
                    // 1. Flip buffer to prepare for reading out bytes
                    buffer.flip();

                    // 2. Decode bytes to UTF-8 text and append to StringBuilder
                    messageBuffer.append(StandardCharsets.UTF_8.decode(buffer).toString());

                    // 3. Clear buffer so it's ready for the next read call
                    buffer.clear();

                    // 4. Process all complete lines (delimited by \n)
                    int newlineIndex;
                    while ((newlineIndex = messageBuffer.indexOf("\n")) != -1) {
                        // Extract a single complete JSON line
                        String jsonLine = messageBuffer.substring(0, newlineIndex).trim();

                        // Remove the processed line from StringBuilder
                        messageBuffer.delete(0, newlineIndex + 1);

                        if (!jsonLine.isEmpty()) {
                            // 5. Parse into test names and run tests
                            String[] testNames = gson.fromJson(jsonLine, String[].class);
                            
                            // Fix type to List<TestResult>
                            List<TestResult> results = runTests(Arrays.asList(testNames));
                            String jsonResponse = gson.toJson(results) + "\n";

                            // Convert UTF-8 string to byte array and send over channel
                            byte[] responseBytes = jsonResponse.getBytes(StandardCharsets.UTF_8);
                            ByteBuffer writeBuffer = ByteBuffer.wrap(responseBytes);
                            while (writeBuffer.hasRemaining()) {
                                clientChannel.write(writeBuffer);
                            }
                        }
                    }
                }
            }
        }
    } finally {
        // Clean up socket file on exit
        Files.deleteIfExists(socketPath);
    }
}
}