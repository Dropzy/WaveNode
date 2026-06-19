const { contextBridge, ipcRenderer } = require('electron');

const configuredBaseUrl =
  process.env.WAVENODE_API_BASE_URL ||
  process.env.MUSIC_SERVER_API_BASE_URL ||
  'http://127.0.0.1:8080/api';

contextBridge.exposeInMainWorld('MUSIC_SERVER_API_BASE_URL', configuredBaseUrl);
contextBridge.exposeInMainWorld('WAVENODE_DESKTOP', true);
contextBridge.exposeInMainWorld('WAVENODE_DESKTOP_BRIDGE', {
  getServerUrl: () => ipcRenderer.invoke('wavenode:get-server-url'),
  setServerUrl: url => ipcRenderer.invoke('wavenode:set-server-url', url),
  discoverServers: () => ipcRenderer.invoke('wavenode:discover-servers'),
});
