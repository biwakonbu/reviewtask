# Repository Guidelines

## Project Structure & Modules
- `cmd/`: Cobra CLI entrypoints and subcommands.
- `internal/`: Core packages (GitHub client, AI processing, task logic).
- `main.go`: Binary bootstrap (`reviewtask`).
- `test/`: Integration and workflow tests (`*_test.go`).
- `scripts/`: Utilities like `run-with-limits.sh`.
- `docs/`, `images/`, `dist/`: Documentation, assets, build artifacts.
- `.pr-review/`: Runtime data (auth, PR cache, tasks). Gitignored.

## Build, Test, Develop
- `make build`: Build local binary `./reviewtask` with version metadata.
- `make test`: Run `go test -v ./...` with conservative CPU/GC limits.
- `make lint`: Run `golangci-lint` via `scripts/run-with-limits.sh`.
- `make ci`: fmt check, vet, lint, tests, build — mirrors local CI.
- `make build-all`: Cross‑compile into `dist/`; `make package` to archive.
Examples:
```bash
GOMAXPROCS=1 TEST_P=1 make test
./reviewtask status --all
```

## Coding Style & Naming
- Language: Go 1.23; format with `gofmt` (CI enforces) and `golangci-lint` (see `.golangci.yml`).
- Indentation: tabs (Go default); imports grouped; keep files `go fmt` clean.
- Packages: lowercase, no underscores; files use snake-like words if needed.
- Exported identifiers: `CamelCase`; unexported: `camelCase` starting lowercase.
- CLI commands live under `cmd/` with clear, action‑oriented names.

## Testing Guidelines
- Framework: standard `testing` with `testify` assertions.
- Location: unit/integration in `test/` and alongside packages.
- Naming: files end with `_test.go`; tests `TestXxx`; table‑driven where sensible.
- Run: `make test` or `go test ./...`. Aim to keep/regressions covered; add cases when changing behavior.

### Golden Tests (Local-Only)
- Purpose: snapshot prompt outputs and CLI templates to catch regressions.
- Default prompt profile is `v2` (rich). Use `--profile legacy` in debug to compare old behavior.
- Run fast set: `make test-fast` (config, AI prompt, CLI prompt)
- Update snapshots intentionally:
  - `UPDATE_GOLDEN=1 go test -v ./internal/ai -run BuildAnalysisPrompt_Golden`
  - `UPDATE_GOLDEN=1 go test -v ./cmd -run PromptStdout_PRReview_Golden`
- Files: `internal/ai/testdata/prompts/<profile>/*.golden`, `cmd/testdata/pr-review.golden`
- No AI/network is used; these tests run offline.

## Commit & PR Guidelines
- Commits: Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`). Keep messages imperative and scoped.
- PRs: include purpose, linked issues, behavior changes, and test coverage. Add usage notes/docs if CLI behavior changes. Run `make ci` locally and attach relevant outputs/logs when debugging.

## Security & Configuration
- Auth: prefer `GITHUB_TOKEN`; local creds stored in `.pr-review/auth.json` (do not commit). 
- Resource limits: use `scripts/run-with-limits.sh` or envs `GOMAXPROCS`, `GOMEMLIMIT`, `GOGC` for heavy commands.
- Secrets: never embed tokens in code, tests, or docs.
