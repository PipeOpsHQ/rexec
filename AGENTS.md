# Repository Guidelines

## Project Structure
Backend Go code lives in `cmd/` (binaries) and `internal/` (packages). Key entrypoints: `cmd/rexec` (API server), `cmd/rexec-cli` (CLI), `cmd/rexec-tui` (Bubble Tea TUI), and `cmd/rexec-agent` (remote agent). The web UI is in `web-ui/` and compiles to static assets in `web/` served by the API. Deployment and dev tooling live in `docker/`, `scripts/`, and `docs/`. Build output goes to `bin/` and `downloads/`.

## Build, Test, and Development
- `make setup`: install Go deps, pull base images, create `/var/lib/rexec` volumes, and seed `.env`.
- `make build` / `make run` / `make dev`: build and run the server locally (hot-reload uses `air`).
- `make cli-all`: build CLI, TUI, and agent tools.
- UI: `cd web-ui && npm install`, `npm run dev` (vite dev server), `npm run build` (writes to `../web/`), `npm run check` (type + Svelte validation).
- Docker stack: `make docker-run` and `make logs`.

## Coding Style & Naming
Go follows standard `gofmt` output (tabs, 1TBS). Run `make fmt` and `make lint` (`golangci-lint`) before pushing. Tests and helpers should be table-driven where it helps readability. Svelte/TypeScript uses 2-space indentation; keep new components in `web-ui/src/lib/components` and stores in `web-ui/src/lib/stores`. Prefer descriptive names like `ContainerManager` over abbreviations.

## Testing Guidelines
Backend tests use the Go `testing` package and live beside code as `*_test.go`. Run `make test` or `make test-coverage`; Docker-dependent tests require a running Docker daemon.

## Commits & Pull Requests
Commit messages follow conventional commits: `feat:`, `fix:`, `refactor:`, `style:`, `docs:`, `chore(deps):` (e.g., `feat: add agent refresh-token`). PRs should explain the why + what, link issues, and include screenshots/gifs for UI changes. Ensure Go tests and, when UI is touched, `npm run check` pass.

