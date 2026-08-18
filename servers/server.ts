import * as net from 'net';
import * as fs from 'fs';
import { runCLI } from '@jest/core';

// 1. Result interface matching Go's runners.TestResult
interface TestResult {
    name: string;
    passed: boolean;
    duration: number; // nanoseconds for Go
    output: string;
}

async function runJestTests(testPaths: string[]): Promise<TestResult[]> {
    try {
        // Call Jest programmatically
        const { results } = await runCLI(
            {
                runInBand: true,
                silent: true,
                _ : testPaths,
                // Fallback transform for TypeScript if target project lacks custom jest config
                transform: JSON.stringify({
                    '^.+\\.(ts|tsx)$': 'ts-jest'
                })
            } as any,
            [process.cwd()]
        );

        const mappedResults: TestResult[] = [];

        for (const testFile of results.testResults) {
            for (const assertion of testFile.testResults) {
                mappedResults.push({
                    name: `${testFile.testFilePath}::${assertion.title}`,
                    passed: assertion.status === 'passed',
                    duration: (assertion.duration || 0) * 1e6, // ms to nanoseconds
                    output: assertion.failureMessages.join('\n')
                });
            }
        }

        return mappedResults;
    } catch (err: any) {
        // Error boundary: Return a failing result instead of letting the server crash
        return testPaths.map((path) => ({
            name: path,
            passed: false,
            duration: 0,
            output: `Daemon Jest Execution Error: ${err?.message || String(err)}`
        }));
    }
}

function main(): void {
    const socketIndex = process.argv.indexOf('--socket');
    if (socketIndex === -1 || !process.argv[socketIndex + 1]) {
        console.error("Missing required --socket argument");
        process.exit(1);
    }
    const socketPath = process.argv[socketIndex + 1];

    // Clean stale socket file
    if (fs.existsSync(socketPath)) {
        fs.unlinkSync(socketPath);
    }

    // Create UNIX socket server
    const server = net.createServer((socket) => {
        let buffer = '';

        socket.on('data', async (chunk) => {
            buffer += chunk.toString('utf-8');

            // Handle line-delimited JSON messages (splits by newline)
            if (buffer.includes('\n')) {
                const lines = buffer.split('\n');
                // Retain incomplete trailing fragment in buffer
                buffer = lines.pop() || '';

                for (const line of lines) {
                    const trimmed = line.trim();
                    if (!trimmed) continue;

                    try {
                        const testPaths: string[] = JSON.parse(trimmed);
                        const results = await runJestTests(testPaths);
                        socket.write(JSON.stringify(results) + '\n');
                    } catch (parseErr: any) {
                        console.error("Malformed JSON payload received:", parseErr);
                    }
                }
            }
        });
    });

    server.listen(socketPath, () => {
        // Handshake for Go's Scanner
        console.log("READY");
    });

    // Cleanup on exit
    process.on('SIGINT', () => cleanup(server, socketPath));
    process.on('SIGTERM', () => cleanup(server, socketPath));
}

function cleanup(server: net.Server, socketPath: string) {
    server.close();
    if (fs.existsSync(socketPath)) {
        fs.unlinkSync(socketPath);
    }
    process.exit(0);
}

main();
//cutting my losses with this server and just doing the llm for this was much better, ill struggle through the java server.