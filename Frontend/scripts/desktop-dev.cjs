const { spawn } = require('node:child_process');
const http = require('node:http');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const viteBin = path.join(root, 'node_modules', 'vite', 'bin', 'vite.js');
const electronBin = path.join(root, 'node_modules', 'electron', 'cli.js');
const devServerUrl = 'http://127.0.0.1:5173';

const children = new Set();

const spawnChild = (command, args) => {
  const child = spawn(command, args, {
    cwd: root,
    stdio: 'inherit',
    env: {
      ...process.env,
      VITE_DEV_SERVER_URL: devServerUrl,
    },
  });
  children.add(child);
  child.on('exit', () => children.delete(child));
  return child;
};

const waitForServer = async (url, timeoutMs = 30000) => {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    const isReady = await new Promise(resolve => {
      const request = http.get(url, response => {
        response.resume();
        resolve(response.statusCode && response.statusCode < 500);
      });
      request.on('error', () => resolve(false));
      request.setTimeout(1000, () => {
        request.destroy();
        resolve(false);
      });
    });

    if (isReady) return;
    await new Promise(resolve => setTimeout(resolve, 300));
  }
  throw new Error(`Timed out waiting for ${url}`);
};

const shutdown = code => {
  for (const child of children) {
    child.kill();
  }
  process.exit(code);
};

process.on('SIGINT', () => shutdown(0));
process.on('SIGTERM', () => shutdown(0));

(async () => {
  const vite = spawnChild(process.execPath, [viteBin, '--host', '127.0.0.1', '--port', '5173', '--strictPort']);
  vite.on('exit', code => {
    if (code) shutdown(code);
  });

  await waitForServer(devServerUrl);
  const electron = spawnChild(process.execPath, [electronBin, '.']);
  electron.on('exit', code => shutdown(code || 0));
})().catch(error => {
  console.error(error.message);
  shutdown(1);
});
