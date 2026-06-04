# AGENTS.md

## PROJECT

**Persephone** — Concurrency-first Git reimagining in Go. CLI binary: `purr`.
Stack: Go 1.25+ · Cobra · flat-file `.purr/objects` · local only

## CRITICAL RULES

- **File naming**: `internal/purrCommands/` uses PascalCase — `Add.go`, `Commit.go`, `Init.go`. Never create lowercase variants. Wrong casing creates shadow files and breaks the build.
- **Module import**: always `Persephone/internal/...` (capital P). `persephone/` is wrong.
- **UI output**: never use lipgloss directly in `cmd/` or `internal/purrCommands/`. Always use `internal/ui` helpers.
  - `ui.Warningf` — execution continues after this
  - `ui.Errorf` — execution stops here, or dangerous state / data corruption
  - `ui.Successf`, `ui.Infof`, `ui.Hintf` — for success, general info, actionable suggestions
- **Comments**: explain why, constraints, invariants, non-obvious behavior. Never restate what code does. Never delete existing comments.
- **Verify with**: `go test -race ./...` — do not build binary unless strictly necessary.

## NON-OBVIOUS COMMANDS

```sh
make dev ARGS="init"    # go run ./cmd/purr <subcommand> — run without building
```

## VERIFIED

Last verified: `2026-06-02` · Go 1.26.3 · Linux
