# WaveNode Frontend

React, TypeScript, Vite, and Electron client surfaces for WaveNode.

## Targets

- Web app served by Docker/nginx or Vite during development
- Standalone Electron desktop app that connects to any WaveNode server

The Electron client uses the same React player as the web app, but it adds a desktop-only login flow for entering a WaveNode server URL and discovering local servers on the LAN.

## Development

Install dependencies:

```bash
npm ci
```

Run the browser app:

```bash
npm run dev
```

Run the Electron desktop client against the local Vite dev server:

```bash
npm run desktop:dev
```

Build the production web bundle:

```bash
npm run build
```

Build the Electron desktop app:

```bash
npm run desktop:build
```

Electron build output is written to `release/`.

## Server Selection

Browser builds use the server they are hosted from, or `VITE_API_BASE_URL` when configured.

Electron builds are standalone clients. On the login screen users can:

- enter a server URL such as `http://192.168.1.70:8080`
- click **Find servers on this network** to discover local WaveNode servers

The selected server is stored locally in the desktop app and used for API, artwork, and streaming URLs.

## Useful Scripts

```bash
npm run build          # Type-check and build the web app
npm run desktop:dev    # Start Vite and Electron together
npm run desktop:build  # Build the desktop package
npm run lint           # Run ESLint
```

## Notes

The Docker frontend image only contains the web bundle served by nginx. Electron is distributed separately as a desktop client package.
