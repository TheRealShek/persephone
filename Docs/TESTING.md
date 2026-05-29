# Persephone Test Suite Guide

This document outlines how to execute and understand the Persephone (`purr`) test suite.

The test suite is built strictly using Go's native testing framework (`testing`) and requires no external mocking libraries or dependencies outside of the core Go toolchain.

## Prerequisites

- Go 1.25 or higher (required for `errors.Join` and other modern stdlib features)
- A POSIX-compliant environment (Linux, macOS) or Windows.
  *(Note: `hidden_windows.go` tests will only execute natively on a Windows OS or CI runner).*

---

## 1. Running All Tests (Standard)

To execute the entire test suite across all packages (Unit, Integration, and E2E):

```bash
go test ./...
```

For verbose output (showing every test case name):

```bash
go test -v ./...
```

---

## 2. Running Concurrency and Race Tests (Critical)

Persephone relies heavily on concurrency (goroutine worker pools) for file processing during `purr add`. Data races can cause catastrophic repository corruption. **Always run the race detector before committing.**

```bash
go test -race ./...
```

This will specifically stress:
- `TestAddAllPurrFiles_ConcurrencyStress`: Adds 100+ files simultaneously to exercise the bounded semaphore.
- Map writes in `Add.go` protected by `sync.Mutex`.

---

## 3. Running Specific Package Tests

If you are only working on a specific feature, you can run tests for just that package to save time.

**Core Utilities & Pure Logic (Tree hashing, Index binary serialization):**
```bash
go test -v ./internal/utils
```

**Commands & File System Integration (Add, Commit):**
```bash
go test -v ./internal/purrCommands
```

**End-to-End CLI Workflows (Init -> Add -> Commit):**
```bash
go test -v ./cmd
```

---

## 4. Test Coverage

To view the percentage of code covered by tests, run the coverage tool:

```bash
# Generate the coverage profile
go test -coverprofile=coverage.out ./...

# View a clean, function-by-function breakdown in the terminal
go tool cover -func=coverage.out

# (Optional) Open an interactive HTML visualization of covered lines
go tool cover -html=coverage.out
```

---

## Key Testing Philosophies to Maintain

When adding new tests, strictly adhere to the following principles:

1. **No Mocking the File System**: Do not use `afero` or other in-memory file systems. A VCS interacts heavily with real OS quirks. Always use `t.TempDir()` (via `testutils.SetupTestRepo(t)`) for integration testing.
2. **Determinism over Real Time**: When generating Git-compatible objects (like Commits), inject a static `time.Time` rather than relying on `time.Now()`. This guarantees stable SHA-1 hashes in tests.
3. **Abuse Edge Cases**: Test binary serialization (`index.go`) with paths exceeding 0xFFF length, misaligned NUL padding, and broken magic headers.
4. **Time-Of-Check to Time-Of-Use (TOCTOU)**: `Add.go` uses broken symlinks intentionally injected mid-execution to verify that the application correctly handles files disappearing between enumeration and hashing.

---

## 5. Installing & Testing in a Real Environment

To verify Persephone's behavior on a real, high-volume folder, you can compile and install it globally to your local user binary directory:

### 5.1 Build and Install
```bash
# Compile and copy directly to your local user binaries directory (standard in Fedora/Linux)
go build -o ~/.local/bin/purr ./cmd/purr
```

Ensure `~/.local/bin` is in your shell's `$PATH` (e.g. via `export PATH=$PATH:$HOME/.local/bin` in your shell configuration).

### 5.2 Manual Verification Walkthrough
Create a temporary folder separate from your codebase and run a full manual VCS lifecycle:

```bash
mkdir -p /tmp/purr-manual-test
cd /tmp/purr-manual-test

# 1. Setup global config
purr config user.name "Your Name"
purr config user.email "you@example.com"

# 2. Initialize repo
purr init

# 3. Create a test file
echo "hello world" > hello.txt

# 4. Stage concurrently
purr add .

# 5. Verify staging index
purr ls

# 6. Commit snapshot
purr commit -m "Manual verification snapshot"
```
