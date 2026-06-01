<!--
  AGENTS.md — Compressed repository memory for AI coding agents.
  Purpose : fast orientation, reduced re-exploration, fewer hallucinated assumptions.
  Contract: contains only verified facts. Prefer file pointers over inline prose.
  Scope   : NOT a behavioral handbook. NOT a replacement for linters or CI.
  Size    : keep under 150 lines. If it grows, move detail to agent-docs/*.
  Updates : only with verified information. If state is uncertain, ask the user
            before editing. Bump the VERIFIED date after any confirmed change.
-->

# AGENTS.md

## PROJECT

**Persephone** — An experimental, concurrency-first reimagining of Git as a modern VCS, built in Go. CLI tool is called `purr`.
Stack: Go 1.25+ (Minimum requirement) · Cobra CLI · flat-file `.purr/objects` (Git-compatible layout) · local only
Shape: monorepo no · single service (CLI binary) · no API (local CLI tool)


## COMMANDS

```sh
# Bootstrap
go mod download

# Build the `purr` binary
make build                              # produces ./purr

# Install to $GOPATH/bin
make install

# Run without building (pass command via ARGS)
make dev ARGS="init"                    # e.g. runs `go run ./cmd/purr init`

# Run all tests
make test                               # go test -v ./...

# Run targeted single-package tests (faster)
go test ./internal/purrCommands/...     # skip full suite when doing isolated work

# Run tests with race detector
go test -race ./...

# Format code
make fmt                                # go fmt ./...

# Clean build artifacts
make clean
```

No env vars required. No external services.


## ARCHITECTURE

- Chose Cobra over `flag` stdlib — provides subcommand routing, help generation, and flag parsing for `purr init`, `purr add`, etc.
- `.purr/` directory structure mirrors Git's `.git/` layout — objects stored as zlib-compressed blobs under `objects/XX/YYYYYY...`, staging via binary `index` file, HEAD as plain text ref pointer
- Concurrency via goroutine worker pool (bounded at `runtime.NumCPU() * 5`) with semaphore channels — applies to `purr add` file hashing and blob writing
- Index uses a Git-inspired binary format: 12-byte header (`DIRC` magic + version 2 + entry count), followed by serialized `IndexEntry` structs. Entry bitfield packing still diverges from Git.
- User config stored globally at `~/.purrconfig` as JSON (not per-repo)
- `internal/purrCommands/` contains command logic; `internal/utils/` contains shared types, index I/O, SHA-1 hashing, and commit object helpers — these two packages are intentionally separated; commands import utils but not vice versa
- `internal/ui/` centralizes all terminal UI components, `lipgloss` styling, and layout formatting. Command packages (`cmd/` and `internal/purrCommands/`) must strictly invoke exported helpers from `ui` rather than defining raw styles or instantiating `lipgloss` directly.
- Tree objects use Git-compatible recursive formats, supporting nested `040000` sub-tree serialization.
- Commit objects use Git-compatible plain-text headers and metadata payloads, compressed with zlib under `.purr/objects`
- `purr log` resolves HEAD and walks the current single-parent commit chain newest-to-oldest; malformed ancestry cycles are rejected
- Command repeat policy: `init` is create-only; `commit` rejects unchanged trees; `add` and `config` are intentionally repeatable mutations; `ls` and `log` are repeatable inspections


## LAYOUT

| Path | Notes |
|---|---|
| `cmd/purr/main.go` | Binary entrypoint — just calls `cmd.Execute()` |
| `cmd/root.go` | Cobra root command definition and flag setup |
| `cmd/{init,add,commit,config,ls,log}.go` | Thin CLI wrappers — each delegates to `internal/purrCommands/` |
| `internal/purrCommands/` | Core command logic: `Init.go`, `Add.go`, `Commit.go`, `Config.go`, `Ls.go`, `Log.go` |
| `internal/ui/` | All UI components, styles, lipgloss logic, and output formatting |
| `internal/utils/types.go` | All shared data types: `IndexEntry`, `PurrConfig`, `TreeEntries`, `CommitObj` |
| `internal/utils/index.go` | Binary index read/write (`ReadIndex`, `WriteIndex`) |
| `internal/utils/commitFunctions.go` | Tree/commit object building and parsing, SHA-1 computation, zlib compression |
| `internal/utils/shaFunctions.go` | SHA-1 blob hashing (`WriteBlobWithSHA`) |
| `internal/utils/config.go` | Global config read/write (`~/.purrconfig`) |
| Docs/ | Design documents: Git internals, limitations, Phase 1 plan, command implementation flows |


## CONVENTIONS

- **Error handling**: Commands print errors to stdout and call `os.Exit(1)` for fatal cases; non-fatal errors are logged with `log.Printf` and skipped
- **Concurrency safety**: All shared map writes in `Add.go` are protected by `sync.Mutex`; the worker pool size is bounded by a semaphore channel
- **File naming**: PascalCase in `internal/purrCommands/` (e.g., `Add.go`, `Commit.go`); camelCase in `internal/utils/` (e.g., `commitFunctions.go`)
- **Index determinism**: Index entries are always sorted alphabetically by path before writing to disk
- **Build & test boundaries**: Do not build the binary unless strictly necessary. Rely primarily on basic test commands (`go test -race ./...`) for routine verification to avoid environment resource bottlenecks.


## GOTCHAS

- **Module name**: The Go module is named `Persephone` (capital P) — imports must use `Persephone/internal/...`, not lowercase
- **Commit format**: Commit objects use Git-style plain-text serialization. `utils.CommitObj` is the in-memory representation; global user config remains JSON.
- **Testing nuances**: The test suite covers E2E deletion scenarios and recursive tree building, but always run `make test` and `go test -race ./...` before committing to catch concurrency regressions.
- **Hidden files**: Both `purr add .` and `purr add <file>` skip files/directories starting with `.` — this is intentional, not a bug
- **Index header**: The `.purr/index` file must have a valid 12-byte header or `ReadIndex` will fail — `purr init` creates this automatically
- **Config location**: `~/.purrconfig` is global, not per-repo — there is no `.purr/config` equivalent

### Comments

Write comments for future maintainers, not code readers.

- Always add comments where design intent, constraints, invariants, trade-offs, or non-obvious behavior would otherwise require reverse-engineering.
- Explain why, constraints, invariants, trade-offs, and non-obvious behavior.
- Do not narrate the implementation or restate the code.
- Remove low-value comments when editing code.
- Prefer fewer high-signal comments over exhaustive coverage.
- Leave self-explanatory code uncommented.

Rule: if a comment does not help a contributor understand the design or safely modify the code, do not write it.

## VERIFIED

Last verified : `2026-06-02`
Verified by   : agent session · `0062185`
Environment   : Linux · Go 1.26.3 (Verified) · make
