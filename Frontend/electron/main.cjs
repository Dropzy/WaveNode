const { app, BrowserWindow, Menu, ipcMain, shell } = require('electron');
const os = require('node:os');
const path = require('node:path');

const isDev = !app.isPackaged;
const devServerUrl = process.env.VITE_DEV_SERVER_URL || 'http://127.0.0.1:5173';
const desktopUserAgent = `WaveNode Desktop/${app.getVersion()} (${process.platform})`;
const defaultServerUrl = process.env.WAVENODE_SERVER_URL || process.env.MUSIC_SERVER_URL || 'http://127.0.0.1:8080';
let savedServerUrl = defaultServerUrl;

const escapeHTML = value => String(value)
  .replaceAll('&', '&amp;')
  .replaceAll('<', '&lt;')
  .replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;')
  .replaceAll("'", '&#39;');

const normalizeServerUrl = value => {
  const trimmed = String(value || '').trim();
  if (!trimmed) {
    return defaultServerUrl;
  }
  const withProtocol = /^https?:\/\//i.test(trimmed) ? trimmed : `http://${trimmed}`;
  return withProtocol.replace(/\/api\/?$/i, '').replace(/\/+$/, '');
};

const privateSubnetPrefixes = () => {
  const prefixes = new Set();
  for (const addresses of Object.values(os.networkInterfaces())) {
    for (const address of addresses || []) {
      if (address.family !== 'IPv4' || address.internal) {
        continue;
      }
      const parts = address.address.split('.');
      if (parts.length !== 4) {
        continue;
      }
      const first = Number(parts[0]);
      const second = Number(parts[1]);
      const isPrivate = first === 10 || (first === 172 && second >= 16 && second <= 31) || (first === 192 && second === 168);
      if (isPrivate) {
        prefixes.add(parts.slice(0, 3).join('.'));
      }
    }
  }
  return [...prefixes];
};

const validateServer = async url => {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 1200);
  try {
    const response = await fetch(`${url}/health`, {
      signal: controller.signal,
      headers: { 'User-Agent': desktopUserAgent },
    });
    const body = await response.text();
    if (!response.ok || !body.toLowerCase().includes('wavenode')) {
      return null;
    }
    const parsed = new URL(url);
    const port = parsed.port ? `:${parsed.port}` : '';
    return {
      name: `WaveNode at ${parsed.hostname}${port}`,
      url,
    };
  } catch {
    return null;
  } finally {
    clearTimeout(timeout);
  }
};

const discoverServers = async () => {
  const candidates = new Set([defaultServerUrl, 'http://127.0.0.1:8080']);
  for (const prefix of privateSubnetPrefixes()) {
    for (let host = 1; host <= 254; host += 1) {
      candidates.add(`http://${prefix}.${host}:8080`);
      candidates.add(`http://${prefix}.${host}`);
    }
  }

  const urls = [...candidates].map(normalizeServerUrl);
  const results = [];
  const concurrency = 48;
  let cursor = 0;

  const workers = Array.from({ length: concurrency }, async () => {
    while (cursor < urls.length) {
      const url = urls[cursor];
      cursor += 1;
      const result = await validateServer(url);
      if (result && !results.some(existing => existing.url === result.url)) {
        results.push(result);
      }
    }
  });

  await Promise.all(workers);
  return results.sort((a, b) => a.url.localeCompare(b.url));
};

const createWindow = () => {
  const mainWindow = new BrowserWindow({
    width: 1280,
    height: 820,
    minWidth: 960,
    minHeight: 640,
    title: 'WaveNode',
    backgroundColor: '#101411',
    show: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  mainWindow.webContents.setUserAgent(desktopUserAgent);

  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    void shell.openExternal(url);
    return { action: 'deny' };
  });

  mainWindow.webContents.on('did-fail-load', (_event, errorCode, errorDescription, validatedURL) => {
    const message = escapeHTML(`WaveNode could not load the desktop UI.\n\n${errorDescription} (${errorCode})\n${validatedURL}`);
    const html = `
      <body style="margin:0;background:#101411;color:#f7fff9;font-family:Segoe UI,Arial,sans-serif;display:grid;place-items:center;height:100vh">
        <main style="max-width:640px;padding:32px">
          <h1 style="margin:0 0 12px;font-size:28px">WaveNode failed to load</h1>
          <p style="white-space:pre-wrap;color:#b8c7be">${message}</p>
        </main>
      </body>`;
    void mainWindow.loadURL(`data:text/html;charset=utf-8,${encodeURIComponent(html)}`);
  });

  mainWindow.webContents.on('render-process-gone', (_event, details) => {
    console.error('WaveNode renderer stopped:', details);
  });

  if (isDev) {
    void mainWindow.loadURL(devServerUrl);
    mainWindow.webContents.openDevTools({ mode: 'detach' });
  } else {
    void mainWindow.loadFile(path.join(__dirname, '../dist/index.html'));
  }
};

app.whenReady().then(() => {
  app.userAgentFallback = desktopUserAgent;
  Menu.setApplicationMenu(null);
  ipcMain.handle('wavenode:get-server-url', () => savedServerUrl);
  ipcMain.handle('wavenode:set-server-url', (_event, url) => {
    savedServerUrl = normalizeServerUrl(url);
    return savedServerUrl;
  });
  ipcMain.handle('wavenode:discover-servers', discoverServers);
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});
