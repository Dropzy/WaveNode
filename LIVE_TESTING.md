# Live Testing

The project includes a local live-test runner for Windows.

## Start

From the repository root:

```powershell
.\scripts\start-live-test.ps1
```

The script:

- starts the Go backend on `http://127.0.0.1:8080`
- starts the Vite frontend on `http://127.0.0.1:5173`
- uses the local PostgreSQL database configured in the script
- creates or refreshes an opt-in test administrator
- waits for both services and verifies login before reporting success

Default login:

```text
Username: live-test-admin
Password: admin123
```

Custom credentials and ports can be supplied:

```powershell
.\scripts\start-live-test.ps1 -Username tester -Password test-password -BackendPort 8080 -FrontendPort 5173
```

## Stop

```powershell
.\scripts\stop-live-test.ps1
```

Backend logs are written to `.cache\live-test`.

The seeded account is only enabled by the live-test script. Normal backend startup does not create or update it.

## Smoke Test

For an automated check that starts the stack, verifies the backend, verifies the frontend, renders the login page in headless Edge when available, and then stops the stack:

```powershell
.\scripts\smoke-live-test.ps1
```
