# Persephone — Project Roadmap & Backlog

This document tracks historical development goals, critical security and structural audit findings, and future feature backlogs for the **Persephone** VCS.

---

## 1. Phase 1 Accomplishments

Our primary Phase 1 objective was to bootstrap the VCS structure, staging index, and basic snapshot storage.

- **On-Disk Layout**: Configured `.purr/` directory with `objects/`, `refs/`, `index`, and `HEAD`.
- **Concurrent Staging (`purr add`)**: Bounded worker goroutine pools for concurrent zlib compression and hash computation. Tracks file deletions and gracefully handles missing files.
- **Atomic Snapshots (`purr commit`)**: Recursive tree and commit object serialization, saving snapshots as content-addressable blobs. Robust error propagation prevents silent staging and zlib compression failures.
- **Basic Configuration**: Implemented global `~/.purrconfig` parsing.

---

## 2. Codebase Audit & Refactoring Backlog

A comprehensive systems audit performed on 2026-05-28 identified critical bugs, structural gaps, and design debt. These are categorized by severity and prioritized for resolution.

### 2.1 Critical Severity (Data Safety & Stability)

*   **[C1] Non-Atomic Index Writes**
    *   *Issue*: If the process is killed mid-write (crash, Ctrl+C, power loss), a half-written index file results.
    *   *Fix*: Write to a `.lock` temp file first, execute `fsync`, and perform an atomic `rename()` replacement.
    *   *Impact*: Same vulnerability exists in `StoreObject`, `UpdateHEAD`, and `WriteConfig`.
*   **[C2] Platform-Specific Syscalls (Resolved)**
    *   *Issue*: Hard type assertion `fileInfo.Sys().(*syscall.Win32FileAttributeData)` inside `utils.go` and raw UTF16 Windows API calls inside `Init.go` cause compilation failure on Linux/Unix systems natively.
    *   *Resolution*: Platform-specific stat extraction and hidden-file behavior now live under `internal/platform/` in files selected by Go build tags.
*   **[C3] Non-Deterministic Commit Hashes (Resolved)**
    *   *Issue*: Calling `time.Now()` multiple times inside hashing and metadata generation causes the computed hash to mismatch the stored commit content.
    *   *Resolution*: `CommitPurrFiles` captures one timestamp in `CommitObj`; hashing and storage both serialize that same value.
*   **[C4] No Index Lock**
    *   *Issue*: Concurrent `purr add` processes running simultaneously will corrupt the index.
    *   *Fix*: Acquire a file-based lock via `.purr/index.lock` during writes.

### 2.2 High Severity (Correctness & Logic)

*   **[H3] Unreliable ModTime Checks**
    *   *Issue*: Change detection only relies on file modification times, which copy/archive actions can easily bypass or spoof.
    *   *Fix*: Compare both `ModTime` and file `Size` as first-pass heuristics.

### 2.3 Medium & Low Severity (Design Debt)

*   **[M2] Divergent Index Format**: Bitfield packing of path lengths and stage values differs from Git's standard `DIRC` layout.
*   **[L1] SHA-1 Cryptographic Collisions**: Document the migration roadmap to SHA-256 for commit hashes.

---

## 3. Phase 2 Opportunities & Future Roadmap

Once the codebase audit is completely resolved, the project will expand into parallel branches, semantic conflict merging, and peer-to-peer data replication.

### 3.1 Inspection Engine
*   **`purr log` (Baseline Implemented)**: Resolves HEAD and walks the current single-parent loose-object chain newest-to-oldest, rejecting malformed ancestry cycles.
*   **`purr log` (Future Graph Traversal)**: Extend history inspection for merge graphs and index commit nodes inside Pebble or Badger DB for instantaneous historical filtering.
*   **`purr status` (Instant Status)**: Build a persistent background daemon via `fsnotify` to listen for real-time filesystem changes, resolving status queries instantly (O(1)) without recursive tree walks.

### 3.2 Branch & State Management
*   **`purr checkout` (Parallel Checkout)**: Switch branches using parallelized file staging and directory cleanups supported by memory-mapped file buffers.
*   **`purr stash`**: Serialize workspace changes concurrently to Pebble DB blocks.

### 3.3 Semantic Merge Engine
*   **`purr merge` (AST Merges)**: Run three-way merges using AST (Abstract Syntax Tree) parsing (using `go/parser` or Tree-Sitter). If Developer A moves a function to the bottom of a file, and Developer B modifies a parameter of that function at the top, resolve the merge automatically without flat text conflicts.

### 3.4 Masterless Peer-to-Peer Synchronization
*   **DHT Peer Discovery**: Auto-discover team members on the same network subnet using a Distributed Hash Table (DHT).
*   **`libp2p` P2P Syncing**: Transfer loose objects or packfiles directly between developer machines over encrypted libp2p nodes without relying on centralized host hubs (GitHub/GitLab).
