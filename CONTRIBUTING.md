# Contributing to WaveNode

Thank you for helping improve WaveNode.

## Before Starting

- Search existing issues before opening a duplicate.
- Use an issue to discuss large features, schema changes, or breaking API changes.
- Never include music files, credentials, tokens, private paths, or generated artwork in a pull request.

## Development Setup

Requirements:

- Go 1.25 or newer
- Node.js 22 or newer
- PostgreSQL 17, or Docker with Docker Compose

Backend:

```bash
cd Backend
go test ./...
go run ./cmd/server
```

Frontend:

```bash
cd Frontend
npm ci
npm run lint
npm run build
npm run dev
```

Full stack:

```bash
cp .env.example .env
docker compose up -d --build
```

## Pull Requests

- Keep changes focused and preserve existing behavior outside the stated scope.
- Add tests for bug fixes and shared behavior.
- Update documentation when configuration, deployment, or user workflows change.
- Confirm backend tests, frontend lint/build, and Docker Compose validation pass.
- Use clear commit messages and describe manual testing in the pull request.

By contributing, you agree that your contribution is licensed under the MIT License.
