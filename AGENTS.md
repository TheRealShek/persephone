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
Stack: Go 1.25+ · Cobra CLI · flat-file `.purr/objects` (Git-compatible layout) · local only
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
- Index uses a Git-compatible binary format: 12-byte header (`DIRC` magic + version 2 + entry count), followed by serialized `IndexEntry` structs
- User config stored globally at `~/.purrconfig` as JSON (not per-repo)
- `internal/purrCommands/` contains command logic; `internal/utils/` contains shared types, index I/O, SHA-1 hashing, and commit object helpers — these two packages are intentionally separated; commands import utils but not vice versa
- Commit objects use JSON serialization (not Git's plain-text format) — see `utils.CommitObj` struct


## LAYOUT

| Path | Notes |
|---|---|
| `cmd/purr/main.go` | Binary entrypoint — just calls `cmd.Execute()` |
| `cmd/root.go` | Cobra root command definition and flag setup |
| `cmd/{init,add,commit,config,ls-files}.go` | Thin CLI wrappers — each delegates to `internal/purrCommands/` |
| `internal/purrCommands/` | Core command logic: `Init.go`, `Add.go`, `Commit.go`, `Config.go`, `LsFiles.go` |
| `internal/utils/types.go` | All shared data types: `IndexEntry`, `PurrConfig`, `TreeEntries`, `CommitObj` |
| `internal/utils/index.go` | Binary index read/write (`ReadIndex`, `WriteIndex`) |
| `internal/utils/commitFunctions.go` | Tree/commit object building, SHA-1 computation, zlib compression |
| `internal/utils/shaFunctions.go` | SHA-1 blob hashing (`WriteBlobWithSHA`) |
| `internal/utils/config.go` | Global config read/write (`~/.purrconfig`) |
| Docs/ | Design documents: Git internals, limitations, Phase 1 plan, command implementation flows |
| `Purr Commands Guide` | User-facing command reference (linked from README) |


## CONVENTIONS

- **Error handling**: Commands print errors to stdout and call `os.Exit(1)` for fatal cases; non-fatal errors are logged with `log.Printf` and skipped
- **Concurrency safety**: All shared map writes in `Add.go` are protected by `sync.Mutex`; the worker pool size is bounded by a semaphore channel
- **File naming**: PascalCase in `internal/purrCommands/` (e.g., `Add.go`, `Commit.go`); camelCase in `internal/utils/` (e.g., `commitFunctions.go`)
- **Index determinism**: Index entries are always sorted alphabetically by path before writing to disk


## GOTCHAS

- **Module name**: The Go module is named `Persephone` (capital P) — imports must use `Persephone/internal/...`, not lowercase
- **No tests yet**: `make test` runs but there are no test files in the repo — `go test ./...` passes vacuously
- **Hidden files**: Both `purr add .` and `purr add <file>` skip files/directories starting with `.` — this is intentional, not a bug
- **Index header**: The `.purr/index` file must have a valid 12-byte header or `ReadIndex` will fail — `purr init` creates this automatically
- **Config location**: `~/.purrconfig` is global, not per-repo — there is no `.purr/config` equivalent


## VERIFIED

Last verified : `2026-05-28`
Verified by   : agent session · `9322d7f`
Environment   : Linux · Go 1.26.3 · make
