import * as net from 'net';
import * as fs from 'fs';
import { runCLI } from '@jest/core';

const swcJestPath = require.resolve('@swc/jest');
const swcTransform = {
    '^.+\\.tsx?$': [swcJestPath, {
        jsc: {
            parser: { syntax: 'typescript', tsx: true },
            target: 'es2022'
        },
        module: { type: 'commonjs' }
    }]
};

function hasJestConfig(projectRoot: string): boolean {
    const configFiles = [
        'jest.config.js', 'jest.config.ts', 'jest.config.mjs', 'jest.config.mts',
        'jest.config.cjs', 'jest.config.cts', 'jest.config.json'
    ];
    if (configFiles.some((file) => fs.existsSync(`${projectRoot}/${file}`))) return true;

    const packagePath = `${projectRoot}/package.json`;
    if (!fs.existsSync(packagePath)) return false;
    try {
        return Boolean(JSON.parse(fs.readFileSync(packagePath, 'utf-8')).jest);
    } catch {
        return false;
    }
}

// 1. Result interface matching Go's runners.TestResult
interface TestResult {
    test_name: string;
    passed: boolean;
    time_taken: number;
    stdout: string;
    stderr: string;
}

async function runJestTests(testPaths: string[]): Promise<TestResult[]> {
    const projectRoot = process.cwd();
    const mappedResults: TestResult[] = [];

    for (const testPath of testPaths) {
        try {
            const options: any = {
                runInBand: true,
                silent: true,
                _: [testPath]
            };
            if (!hasJestConfig(projectRoot)) {
                options.config = JSON.stringify({
                    rootDir: projectRoot,
                    testEnvironment: 'node',
                    transform: swcTransform
                });
            }

            const { results } = await runCLI(options, [projectRoot]);
            for (const testFile of results.testResults) {
                for (const assertion of testFile.testResults) {
                    mappedResults.push({
                        test_name: `${testFile.testFilePath}::${assertion.title}`,
                        passed: assertion.status !== 'failed',
                        time_taken: (assertion.duration || 0) * 1e6,
                        stdout: '',
                        stderr: assertion.failureMessages.join('\n')
                    });
                }

                if (testFile.testResults.length === 0 && testFile.failureMessage) {
                    mappedResults.push({
                        test_name: testFile.testFilePath,
                        passed: false,
                        time_taken: 0,
                        stdout: '',
                        stderr: testFile.failureMessage
                    });
                }
            }
        } catch (err: any) {
            mappedResults.push({
                test_name: testPath,
                passed: false,
                time_taken: 0,
                stdout: '',
                stderr: `Daemon Jest Execution Error: ${err?.message || String(err)}`
            });
        }
    }

    return mappedResults;
}

function main(): void {
    const socketIndex = process.argv.indexOf('--socket');
    if (socketIndex === -1 || !process.argv[socketIndex + 1]) {
        console.error("Missing required --socket argument");
        process.exit(1);
    }
    const socketPath = process.argv[socketIndex + 1];
    const projectRootIndex = process.argv.indexOf('--project-root');
    const projectRoot = projectRootIndex >= 0 ? process.argv[projectRootIndex + 1] : process.cwd();
    process.chdir(projectRoot);

    const useTcp = process.platform === 'win32';

    // Clean stale socket file
    if (!useTcp && fs.existsSync(socketPath)) {
        fs.unlinkSync(socketPath);
    }

    // Create UNIX socket server
    const server = net.createServer((socket) => {
        let buffer = '';
        let requestQueue = Promise.resolve();

        socket.on('data', (chunk) => {
            buffer += chunk.toString('utf-8');

            // Handle line-delimited JSON messages (splits by newline)
            if (buffer.includes('\n')) {
                const lines = buffer.split('\n');
                // Retain incomplete trailing fragment in buffer
                buffer = lines.pop() || '';

                for (const line of lines) {
                    const trimmed = line.trim();
                    if (!trimmed) continue;

                    requestQueue = requestQueue.then(async () => {
                        let results: TestResult[];
                        try {
                            const testPaths: string[] = JSON.parse(trimmed);
                            if (!Array.isArray(testPaths) || !testPaths.every((path) => typeof path === 'string')) {
                                throw new Error('request must be a JSON array of test paths');
                            }
                            results = await runJestTests(testPaths);
                        } catch (error: any) {
                            results = [{
                                test_name: '<daemon>',
                                passed: false,
                                time_taken: 0,
                                stdout: '',
                                stderr: error?.message || String(error)
                            }];
                        }
                        //if (!socket.destroyed) socket.write(JSON.stringify(results) + '\n');
                    });
                }
            }
        });
    });

    server.listen(useTcp ? { host: '127.0.0.1', port: 0 } : socketPath, () => {
        // Handshake for Go's Scanner
        const address = server.address();
        if (useTcp && address && typeof address !== 'string') {
            console.log(`READY TCP 127.0.0.1:${address.port}`);
        } else {
            console.log("READY");
        }
    });

    // Cleanup on exit
    process.on('SIGINT', () => cleanup(server, socketPath));
    process.on('SIGTERM', () => cleanup(server, socketPath));
}

function cleanup(server: net.Server, socketPath: string) {
    server.close();
    if (process.platform !== 'win32' && fs.existsSync(socketPath)) {
        fs.unlinkSync(socketPath);
    }
    process.exit(0);
}

main();
//cutting my losses with this server and just doing the llm for this was much better, ill struggle through the java server.