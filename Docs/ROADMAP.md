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

*   **[C2] Platform-Specific Syscalls (Resolved)**
    *   *Issue*: Hard type assertion `fileInfo.Sys().(*syscall.Win32FileAttributeData)` and raw UTF16 Windows API calls cause compilation failure on Linux/Unix systems natively.
    *   *Resolution*: Platform-specific stat extraction and hidden-file behavior now live under `internal/platform/` in files selected by Go build tags.
*   **[C3] Non-Deterministic Commit Hashes (Resolved)**
    *   *Issue*: Calling `time.Now()` multiple times inside hashing and metadata generation causes the computed hash to mismatch the stored commit content.
    *   *Resolution*: `CommitPurrFiles` captures one timestamp in `CommitObj`; hashing and storage both serialize that same value.

### 2.3 Medium & Low Severity (Design Debt)

*   **[M2] Divergent Index Format**: Bitfield packing of path lengths and stage values differs from Git's standard `DIRC` layout.
*   **[L1] SHA-1 Cryptographic Collisions**: Document the migration roadmap to SHA-256 for commit hashes.

---

## 3. Phase 2 Opportunities & Future Roadmap

Once the codebase audit is completely resolved, the project will expand into branch management, workspaces state stashing, and conflict resolution before moving toward peer-to-peer data replication.

### 3.1 Inspection Engine

*   **`purr status`**: Displays the active workspace changes categorized by staged, unstaged, and untracked files. The key design note is to build a persistent background daemon via `fsnotify` to listen for real-time filesystem changes, enabling status queries to resolve instantly in $O(1)$ time without requiring heavy recursive tree walks. There are no hard command dependencies.
*   **`purr diff`**: Computes and shows line-level modifications between the working directory, the staging index, and previous commits. The key design note is to utilize Myers diff algorithm optimized with memory-mapped file buffers for fast comparison of large files. It has a hard dependency on the change detection logic implemented for `purr status`.
*   **`purr log` (Future Graph Traversal)**: Extends the history inspection command to support merge graphs, rendering visual branch topologies and indexing commit nodes inside Pebble or Badger DB for instantaneous historical filtering. It has a hard dependency on branch management commands being implemented.

### 3.2 Branch & State Management

*   **`purr branch`**: Creates, deletes, and lists development branches by managing reference files under `.purr/refs/heads/`. The key design note is to enforce atomic ref updates using lockfiles to prevent concurrent branch state corruption. It has a hard dependency on basic reference management.
*   **`purr checkout`**: Switches the workspace to a target branch by updating the working directory and staging index to match the commit tip of the target branch. The key design note is to run file staging and directory cleanups in parallel using a bounded worker pool to maximize checkout speed. It has a hard dependency on `purr branch` to establish branches before checkout.
*   **`purr stash`**: Temporarily saves and resets dirty workspace modifications (both staged and unstaged) to restore a clean working state. The key design note is to serialize workspace changes concurrently to Pebble DB blocks to avoid polluting the main commit ancestry. It has a hard dependency on the index-writing mechanics of `purr checkout` and `purr status`.

### 3.3 Semantic Merge Engine

*   **`purr merge`**: Resolves changes from different branches by performing three-way merges. The key design note is to run three-way merges using Abstract Syntax Tree (AST) parsing (via `go/parser` or Tree-Sitter) to merge changes at the syntax node level rather than simple flat text lines, which auto-resolves reordering and formatting conflicts. It has a hard dependency on `purr branch` and `purr checkout` for managing and switching between merge states.

### 3.4 Masterless Peer-to-Peer Synchronization

*   **DHT Peer Discovery**: Auto-discovers other team members on the same network subnet using a Distributed Hash Table (DHT) without centralized discovery servers. The key design note is to implement local mDNS and DHT routing tables for decentralized node lookup. It has a hard dependency on network connectivity.
*   **`libp2p` P2P Syncing**: Transfers loose objects or packfiles directly between developer machines over encrypted libp2p nodes. The key design note is to implement block-level data syncing using custom libp2p protocols to enable offline-first collaboration. It has a hard dependency on the DHT peer discovery layer.
