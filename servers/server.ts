import * as net from 'net';
import * as fs from 'fs';
import { spawn } from 'child_process';
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
    timed_out?: boolean;
    time_taken: number;
    stdout: string;
    stderr: string;
}

function isTimeoutMessage(message: string): boolean {
    const normalized = message.toLowerCase();
    return normalized.includes('timed out') || normalized.includes('timeout');
}

async function executeJestTest(testPath: string, projectRoot: string, timeoutMs: number): Promise<TestResult[]> {
    const mappedResults: TestResult[] = [];

    try {
        const options: any = {
            runInBand: true,
            silent: true,
            reporters: [],
            watch: false,
            testTimeout: timeoutMs,
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
                    stderr: assertion.failureMessages.join('\n'),
                    timed_out: assertion.status === 'failed'
                        && isTimeoutMessage(assertion.failureMessages.join('\n'))
                });
            }

            if (testFile.testResults.length === 0 && testFile.failureMessage) {
                mappedResults.push({
                    test_name: testFile.testFilePath,
                    passed: false,
                    time_taken: 0,
                    stdout: '',
                    stderr: testFile.failureMessage,
                    timed_out: isTimeoutMessage(testFile.failureMessage)
                });
            }
        }
    } catch (err: any) {
        mappedResults.push({
            test_name: testPath,
            passed: false,
            time_taken: 0,
            stdout: '',
            stderr: `Daemon Jest Execution Error: ${err?.message || String(err)}`,
            timed_out: isTimeoutMessage(err?.message || String(err))
        });
    }

    return mappedResults;
}

function failedTimedOutTest(testPath: string, timeoutMs: number, stdout: string, stderr: string, elapsedNs: number): TestResult[] {
    return [{
        test_name: testPath,
        passed: false,
        timed_out: true,
        time_taken: elapsedNs,
        stdout,
        stderr: `test timed out after ${timeoutMs}ms\n${stderr}`
    }];
}

function isTestResult(value: unknown): value is TestResult {
    if (typeof value !== 'object' || value === null) return false;
    return 'test_name' in value
        && typeof value.test_name === 'string'
        && 'passed' in value
        && typeof value.passed === 'boolean'
        && 'time_taken' in value
        && typeof value.time_taken === 'number'
        && 'stdout' in value
        && typeof value.stdout === 'string'
        && 'stderr' in value
        && typeof value.stderr === 'string'
        && (!('timed_out' in value) || typeof value.timed_out === 'boolean');
}

async function runJestTestWithWatchdog(testPath: string, projectRoot: string, timeoutMs: number): Promise<TestResult[]> {
    const startedAt = process.hrtime.bigint();
    const tsxCLI = require.resolve('tsx/cli');
    const serverScript = process.argv[1];
    if (!serverScript) {
        throw new Error('Cannot resolve the JavaScript test daemon script path');
    }

    return new Promise((resolve) => {
        const child = spawn(process.execPath, [
            tsxCLI,
            serverScript,
            '--run-selector',
            testPath,
            '--project-root',
            projectRoot,
            '--timeout-ms',
            String(timeoutMs)
        ], {
            cwd: projectRoot,
            windowsHide: true,
            stdio: ['ignore', 'pipe', 'pipe']
        });
        let stdout = '';
        let stderr = '';
        let timedOut = false;
        let settled = false;

        child.stdout.setEncoding('utf8');
        child.stderr.setEncoding('utf8');
        child.stdout.on('data', (chunk: string) => { stdout += chunk; });
        child.stderr.on('data', (chunk: string) => { stderr += chunk; });

        const watchdog = setTimeout(() => {
            timedOut = true;
            child.kill('SIGKILL');
        }, timeoutMs);

        const finish = (results: TestResult[]) => {
            if (settled) return;
            settled = true;
            clearTimeout(watchdog);
            resolve(results);
        };

        child.on('error', (error) => {
            finish([{
                test_name: testPath,
                passed: false,
                time_taken: Number(process.hrtime.bigint() - startedAt),
                stdout,
                stderr: `could not start Jest test process: ${error.message}`
            }]);
        });

        child.on('close', (code, signal) => {
            const elapsedNs = Number(process.hrtime.bigint() - startedAt);
            if (timedOut) {
                finish(failedTimedOutTest(testPath, timeoutMs, stdout, stderr, elapsedNs));
                return;
            }

            const marker = 'SEAPIG_RESULT:';
            const resultLine = stdout.split(/\r?\n/).reverse().find((line) => line.startsWith(marker));
            if (resultLine) {
                try {
                    const parsed: unknown = JSON.parse(resultLine.slice(marker.length));
                    if (Array.isArray(parsed) && parsed.every(isTestResult)) {
                        finish(parsed);
                        return;
                    }
                    stderr += '\nInvalid Jest test result: response did not contain an array of test results';
                } catch (error) {
                    stderr += `\nInvalid Jest test result: ${error instanceof Error ? error.message : String(error)}`;
                }
            }

            finish([{
                test_name: testPath,
                passed: false,
                time_taken: elapsedNs,
                stdout,
                stderr: stderr || `Jest test process exited without results (code ${code}, signal ${signal})`
            }]);
        });
    });
}

async function runJestTests(testPaths: string[], timeoutMs: number): Promise<TestResult[]> {
    const projectRoot = process.cwd();
    const mappedResults: TestResult[] = [];

    for (const testPath of testPaths) {
        mappedResults.push(...await runJestTestWithWatchdog(testPath, projectRoot, timeoutMs));
    }

    return mappedResults;
}

async function runJestSelectorFromChild(): Promise<void> {
    const selectorIndex = process.argv.indexOf('--run-selector');
    const projectRootIndex = process.argv.indexOf('--project-root');
    const timeoutIndex = process.argv.indexOf('--timeout-ms');
    const testPath = process.argv[selectorIndex + 1];
    const projectRoot = projectRootIndex >= 0 ? process.argv[projectRootIndex + 1] : process.cwd();
    const timeoutMs = timeoutIndex >= 0 ? Number(process.argv[timeoutIndex + 1]) : 600_000;
    if (!testPath) {
        throw new Error('Missing test selector for isolated Jest process');
    }

    process.chdir(projectRoot);
    const results = await executeJestTest(testPath, projectRoot, timeoutMs);
    process.stdout.write(`SEAPIG_RESULT:${JSON.stringify(results)}\n`, () => process.exit(0));
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
    const timeoutIndex = process.argv.indexOf('--timeout-ms');
    const timeoutMs = timeoutIndex >= 0
        ? Number(process.argv[timeoutIndex + 1])
        : 600_000;
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
                            results = await runJestTests(testPaths, timeoutMs);
                        } catch (error: any) {
                            results = [{
                                test_name: '<daemon>',
                                passed: false,
                                time_taken: 0,
                                stdout: '',
                                stderr: error?.message || String(error)
                            }];
                        }
                        if (!socket.destroyed) {
                            socket.write(JSON.stringify(results) + '\n');
                        }
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

if (process.argv.includes('--run-selector')) {
    runJestSelectorFromChild().catch((error: unknown) => {
        console.error(error instanceof Error ? error.message : String(error));
        process.exitCode = 1;
    });
} else {
    main();
}
//cutting my losses with this server and just doing the llm for this was much better, ill struggle through the java server.