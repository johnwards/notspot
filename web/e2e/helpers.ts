import { execSync, spawn, type ChildProcess } from 'child_process';
import { resolve } from 'path';

const rootDir = resolve(__dirname, '..', '..');
const binary = resolve(rootDir, 'build', 'hubspot');

let serverProcess: ChildProcess | null = null;

export function buildServer(): void {
  execSync('make build', { cwd: rootDir, stdio: 'pipe' });
}

export function startServer(port: number): Promise<void> {
  return new Promise((resolve, reject) => {
    serverProcess = spawn(binary, [], {
      env: {
        ...process.env,
        NOTSPOT_ADDR: `:${port}`,
        NOTSPOT_DB: ':memory:',
      },
      stdio: 'pipe',
    });

    const timeout = setTimeout(() => {
      reject(new Error('Server failed to start within 10s'));
    }, 10000);

    serverProcess.stderr?.on('data', (data: Buffer) => {
      const msg = data.toString();
      if (msg.includes('starting notspot server')) {
        clearTimeout(timeout);
        // Give the server a moment to bind
        setTimeout(() => resolve(), 100);
      }
    });

    serverProcess.on('error', (err) => {
      clearTimeout(timeout);
      reject(err);
    });

    serverProcess.on('exit', (code) => {
      if (code !== null && code !== 0) {
        clearTimeout(timeout);
        reject(new Error(`Server exited with code ${code}`));
      }
    });
  });
}

export function stopServer(): void {
  if (serverProcess) {
    serverProcess.kill('SIGTERM');
    serverProcess = null;
  }
}
