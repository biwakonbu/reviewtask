# Repository Guidelines

## Documentation Management Rules

### Public Documentation Standards
- **User documentation must only describe implemented, production-ready features**
- **Never include future plans, roadmaps, or "coming soon" features in user guides**
- **Internal development documents must not appear in public documentation site**
- **Maintain strict separation between user-facing and developer-internal content**

### Documentation Site Configuration (mkdocs.yml)
- **User Guide**: Include only features available in current release
- **Developer Guide**: Include architecture, setup, testing, contributing (public-appropriate)
- **Exclude from Navigation**:
  - Product Requirements Documents (prd.md)
  - Implementation progress tracking (implementation-progress.md)
  - Internal prompt templates documentation
  - Any documents with "Future", "Planned", or "TODO" sections

### Documentation Content Rules
- **Write about what EXISTS, not what is PLANNED**
- **Remove all "Future Enhancements" sections from user documentation**
- **Keep implementation details and roadmaps in separate internal docs**
- **Verify all documented features are actually implemented before publishing**

### When Adding Documentation
1. Check if feature is fully implemented and tested
2. Place in appropriate section (user-guide vs developer-guide)
3. Exclude internal planning documents from mkdocs.yml
4. Review for any future-tense promises or unimplemented features

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

## Command Safety
- One command per step: never chain with `&&`, `;`, `|`, or multi-line blocks.
  - Bad: `git add -A && git commit -m "msg" && git push`
  - Good (numbered steps, each in its own fence):
    1) `git add -A`
    2) `git commit -m "msg"`
    3) `git push`
- Presentation rules for docs/PRs:
  - Use numbered lists; put exactly one command in each fenced block.
  - State working directory and preconditions before the block (e.g., `repo root`, branch name).
  - Mention expected effect/output briefly after the block.
- Safety rules:
  - Avoid destructive commands (`rm -rf`, `git reset --hard`) in inline guidance. If unavoidable, require explicit confirmation steps and provide rollback instructions.
  - Prefer `make` targets for multi-step workflows instead of chained shell lines (e.g., `make test-fast`).
  - Split OS-specific commands into separate blocks (Linux/macOS/Windows) — do not combine.
- Review policy: Chained or copy-paste multi-line shell snippets should be requested for change before merging.
