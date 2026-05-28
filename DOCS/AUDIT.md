# Purr Codebase Audit

**Date:** 2026-05-28 · **Scope:** Full codebase · **Reviewer:** Systems audit

---

## CRITICAL — Will corrupt data or crash in production

### C1. `PopulateAllIndexField` hardcodes Windows, panics on Linux

**File:** `utils/utils.go:83`

```go
stat := fileInfo.Sys().(*syscall.Win32FileAttributeData)  // line 83
```

This is a hard type assertion to a Windows-only struct. On Linux (your actual OS), this **panics immediately** on every `purr add` call. You cannot be running this code successfully right now unless something else is catching it.

**Fix:** Use build tags (`//go:build`) with platform-specific files. On Linux, use `syscall.Stat_t`. On Windows, use the current code. The `Init.go` file already has Windows-specific `syscall` calls (line 29-47) but doesn't use build tags either — same problem.

---

### C2. Non-atomic index writes — interrupted write = corrupted repo

**File:** `utils/index.go:136`

```go
os.WriteFile(indexPath, buf.Bytes(), 0644)
```

If the process is killed mid-write (crash, Ctrl+C, power loss), you get a half-written index file. Next `ReadIndex` will either fail or read garbage entries. **This is the #1 data loss vector in the entire codebase.**

Git writes to a `.lock` temp file, then does an atomic `rename()`. You must do the same:
```
write to .purr/index.lock → fsync → rename to .purr/index
```

Same problem exists in `StoreObject` (utils.go:74), `UpdateHEAD` (utils.go:143-147), and `WriteConfig` (config.go:62).

---

### C3. Commit hash is non-deterministic — breaks content-addressable guarantee

**File:** `commitFunctions.go:112`

```go
timestamp := time.Now().Unix()  // called inside BuildCommitObject
```

`ComputeCommitSHA1` calls `BuildCommitObject` which calls `time.Now()`. Then `CommitPurrFiles` calls `BuildCommitObject` **again** to get the actual bytes to store. Two calls to `time.Now()` = two different timestamps = **the stored commit doesn't match its hash**.

**Fix:** Generate the timestamp once in `CommitPurrFiles`, inject it into `CommitObj`, and use it in `BuildCommitObject`. Never call `time.Now()` inside a function that is also used for hash computation.

---

### C4. No index lock — concurrent `purr add` corrupts the index

Two `purr add` processes running simultaneously will both read the index, both modify it in memory, and both write it back. Last writer wins, first writer's changes are silently lost.

**Fix:** File-based locking via `.purr/index.lock`. Acquire before read, release after write. Same pattern Git uses.

---

## HIGH — Broken logic or incorrect assumptions

### H1. Flat tree object — no directory hierarchy

**File:** `Commit.go:128-145`

Every indexed file becomes a direct child of the root tree, using the full relative path as its name:
```go
entry := &utils.TreeEntries{
    Name: indexEntry.Path,  // e.g., "src/pkg/main.go"
}
```

Git tree objects represent **one directory level**. A file at `src/pkg/main.go` should produce three nested trees: root → `src` → `pkg` → blob. Your flat tree means:
- You can never reconstruct directory structure from a tree object
- Tree hashes are incompatible with Git
- Future `checkout` implementation becomes impossible without rewriting the tree format

**Fix:** Build a recursive tree builder that groups entries by directory, creates subtrees bottom-up, stores each subtree as a separate object, and references them via `040000` mode entries.

---

### H2. `addAllPurrFiles` silently swallows index read errors

**File:** `Add.go:52`

```go
IndexEntries, _ := utils.ReadIndex(...)  // error discarded
```

If the index is corrupted, you proceed with an empty slice, re-hash everything, and overwrite the (possibly partially valid) index. Silent data loss.

**Fix:** Return the error. If the index can't be read, the user needs to know.

---

### H3. mtime-only change detection is unreliable

**File:** `Add.go:90`

```go
if fileInfo.ModTime().Equal(existingEntry.Mtime) {
    return  // skip
}
```

mtime can be preserved by file copies, archive extraction, or `touch -t`. Content may differ but mtime matches — file silently skipped. Conversely, mtime changes on `chmod` even though content is identical — unnecessary re-hash.

Git uses mtime as a **first-pass heuristic** but falls back to re-hashing when stat data is ambiguous (size change, ctime mismatch, etc.). You should at minimum also check `Size`. Comparing the existing SHA against a freshly computed SHA is the only correct approach for correctness-critical paths.

---

### H4. `CheckConfigFile` returns nil error on validation failure

**File:** `commitFunctions.go:224-234`

```go
if config.UserName == "" {
    return "", "", err  // err is nil here (ReadConfig succeeded)
}
```

When `UserName` is empty, `err` is `nil` from the successful `ReadConfig` call. The caller checks `if err != nil` and proceeds with empty strings. You need to return an explicit error:
```go
return "", "", fmt.Errorf("user.name is not set")
```

---

### H5. Duplicate/dead code for HEAD and branch operations

Both `GetHEADCommit()` and `GetParentCommit()` do the same thing. Both `UpdateHEAD()` and `UpdateBranchRef()` do the same thing. This will inevitably drift and create bugs when one gets updated but not the other.

| Used in Commit flow | Dead / unused |
|---|---|
| `GetHEADCommit()` | `GetParentCommit()` |
| `UpdateHEAD()` | `UpdateBranchRef()` |

**Fix:** Delete the dead functions. One implementation per responsibility.

---

### H6. `Init.go` won't compile on Linux

**File:** `Init.go:9` — imports `syscall`, then uses `syscall.UTF16PtrFromString`, `syscall.GetFileAttributes`, `syscall.SetFileAttributes`, and `syscall.FILE_ATTRIBUTE_HIDDEN` which only exist on Windows.

This code will **not compile** on Linux at all. Combined with C1, the entire project cannot build on Linux without modification.

**Fix:** Split into `init_windows.go` and `init_unix.go` with build tags.

---

## MEDIUM — Design flaws that block future development

### M1. No `purr rm` / no file deletion tracking

If you delete a file from disk and run `purr add .`, the file remains in the index because `WalkAndAddFiles` only visits existing files. Commits will continue to reference deleted files indefinitely.

---

### M2. Index format diverges from Git's actual format

Your metadata block is **62 bytes** (using `int64` for timestamps = 8 bytes each). Git's actual index entry is **62 bytes** too, but uses split `sec`/`nsec` uint32 pairs (4+4=8). Your sizes happen to match but the field layout doesn't. Your `Stage` field is written as a standalone `uint16`, but in Git it's packed into a bitfield with the path length. This means your index files are **not readable by Git** despite the `DIRC` header.

This is fine if you don't care about Git compatibility. But you have the `DIRC` header and version 2, which is actively misleading.

---

### M3. `Size` field is `uint32` — max 4GB files

**File:** `types.go:14`

Files larger than 4GB will have their size truncated. Git has the same limitation in its index format, but your docs talk about improving on Git's limitations. This should be a conscious design decision documented somewhere.

---

### M4. `StoreObject` overwrites existing objects

**File:** `utils.go:74`

```go
return os.WriteFile(objectPath, data, 0644)
```

Content-addressable storage means if the hash exists, the content is identical. You should skip the write if the file exists. This is a correctness issue for crash safety (partial overwrite of a valid object) and a performance issue for large repos.

---

### M5. Tree uses `Name` (full path) but sorts by it — incorrect for nested trees

**File:** `commitFunctions.go:33`

Sorting by full path (`src/a.go` vs `src/b/c.go`) produces a different order than Git's tree sorting, which sorts within each directory level. When you implement proper nested trees, this sort will produce wrong hashes.

---

## LOW — Design debt

| # | Issue | Impact |
|---|---|---|
| L1 | `addAllPurrFiles` returns `error` but `AddPurrFiles` ignores it (line 38) | Errors silently dropped |
| L2 | `runtime.NumCPU() * 5` workers for file I/O is likely too many — disk I/O is the bottleneck, not CPU | Filesystem contention, slower on HDDs |
| L3 | No `.purrignore` equivalent | Can't exclude `node_modules`, build dirs, etc. |
| L4 | SHA-1 is cryptographically broken (known collision attacks) | Fine for Phase 1, but document the decision to migrate to SHA-256 later |
| L5 | No index checksum — can't detect bit-rot or silent corruption | Git appends a SHA-1 of the entire index file |
| L6 | `TreeEntries.Name` vs `TreeEntries.Filename` — both populated, `Filename` never used | Dead field, confusing |
| L7 | No repo root discovery — all commands assume CWD is repo root | `purr add` from a subdirectory will break |

---

## Summary by Severity

| Severity | Count | Blocks |
|---|---|---|
| **CRITICAL** | 4 | Build fails on Linux, data corruption on crash, wrong commit hashes |
| **HIGH** | 6 | Broken tree format, silent errors, dead code |
| **MEDIUM** | 5 | No delete tracking, compatibility lies, overwrite risks |
| **LOW** | 7 | Design debt, missing features |

## Recommended Fix Order

1. **C1 + H6** — Platform build tags (you literally can't build this on Linux right now)
2. **C3** — Fix non-deterministic commit hashing (content-addressable storage is broken)
3. **C2** — Atomic writes with temp+rename pattern
4. **H1** — Recursive tree builder (this is a design rewrite, do it before more commits pile up)
5. **H4 + H2** — Error handling fixes (quick wins)
6. **C4** — Index file locking
7. **H5** — Delete dead functions
8. Everything else as you encounter it
