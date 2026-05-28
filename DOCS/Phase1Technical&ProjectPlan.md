# Go Git Clone (Persephone) — Phase 1: Technical & Project Plan

This document outlines the core architectural specifications and execution roadmap for Phase 1 of **Persephone** (a high-performance, concurrent version control implementation in Go).

---

## Architecture Overview

### On-Disk Structure (`.purr/`)

Persephone mirrors standard Git layouts but operates on highly optimized, concurrent-safe primitives.

```
.purr/
├── objects/              # Content-addressable storage (blobs, trees, commits)
│   ├── ab/               # First two characters of object hash
│   │   └── 12345...      # Compressed binary payload (remaining 38 chars of hash)
│   └── cd/
├── refs/
│   └── heads/
│       └── main          # Branch reference files (contain target commit hash)
├── index                 # Staging area binary/JSON file (serialized metadata)
├── HEAD                  # Active reference pointer (e.g., "ref: refs/heads/main")
└── logs/
    └── HEAD              # Commit history log file
```

### Core Data Models

| Model | Purpose | Cryptographic Hash | Serialization / Details |
| :--- | :--- | :--- | :--- |
| **Blob** | Represents file content | `SHA-1` of file contents | Stored as a compressed binary file under `.purr/objects/XX/` |
| **Tree** | Represents directory state | `SHA-1` of serialized entries | Maps names to modes and child hashes: `{name, mode, hash}` |
| **Commit** | Represents a project snapshot | `SHA-1` of commit metadata | Tracks metadata: `{tree_hash, parent_hash, author, message, timestamp}` |
| **Index** | Manages the staging area | *None (Serialized state)* | Maps files to metadata: `filepath` → `{file_hash, mode, timestamp}` |

---

## 👥 Task Breakdown by Role

To parallelize initial bootstrapping, development is split into modular domains.

### Domain A: Object Storage & Core Cryptography

Focuses on the low-level data layer, content-addressable storage, and highly concurrent I/O operations.

| Task Description | Core Implementation Details | Est. Time |
| :--- | :--- | :--- |
| **1. Initialize Repository** | Create `.purr/` folder and subfolders; write default `HEAD`, empty `index`, and basic configuration files. | `2 hours` |
| **2. Concurrent Blob Storage** | Multi-threaded file reading, SHA-1 generation, and zlib-compressed object writing using a managed pool of worker goroutines. | `4 hours` |
| **3. Parallel Tree Generation** | Concurrently walk directory structures; resolve directory hierarchies into Tree objects in parallel; synchronize using `sync.WaitGroup`. | `5 hours` |
| **4. Thread-Safe Commits** | Read and write immutable commit snapshots atomically using `sync.Mutex` locks to prevent race conditions during updates. | `4 hours` |

**Domain A Subtotal**: `~15 hours`

---

### Domain B: CLI Commands & Staging Management

Focuses on user interaction, state serialization, arguments, and command orchestration.

| Task Description | Core Implementation Details | Est. Time |
| :--- | :--- | :--- |
| **1. CLI & Parser Engine** | Setup CLI interface using Cobra/Flag to parse `purr add`, `purr commit`, and `purr revert` options. | `2 hours` |
| **2. Staging Management (`add`)** | Concurrently scan workspace files; utilize Domain A's concurrent hashing engine; update the staging `index` atomically. | `5 hours` |
| **3. Commit Execution (`commit`)** | Convert current index state to Tree objects (invoking Domain A's tree builder); write Commit object; update `HEAD`. | `4 hours` |
| **4. History Reversion (`revert`)** | Walk parent commits; restore file states concurrently across goroutines; record new reversion snapshots. | `4 hours` |
| **5. Logger & Diagnostic Layer** | Develop structured, thread-safe stdout/stderr writers using buffered channels to prevent write interleaving. | `2 hours` |

**Domain B Subtotal**: `~17 hours`

---

## 🔄 Concurrency Contracts & Integration Points

To ensure a seamless merge, both domains agree on the following boundaries:

- **Thread-Safe Staging Index**: The `.purr/index` is protected by a write-ahead locking protocol. Multiple goroutines can read index state, but writing staged updates requires exclusive locks.
- **Worker Pools**: To prevent thrashing the filesystem, the maximum number of concurrent files read or written simultaneously is bounded by the host's logic processor count (`runtime.NumCPU()`).
- **DAG Soundness**: Reversion, checkout, and tree building must traverse the DAG safely without creating cyclic references.

---

## 📅 Phase 1 Execution Checklist

### Phase 1: Setup & Core Architecture
- [ ] Initialize Go workspace and directory tree layouts.
- [ ] Establish standard concurrency design system (Mutex strategies vs Channel pipelines).
- [ ] Define the exact binary/JSON schema for the serialization of `.purr/index`.
- [ ] **Sync Point**: Locked-in storage protocols and interface boundaries.

### Phase 2: Storage Layer & Shell Skeleton
- [ ] **Domain A**: Complete concurrent blob compression and tree generators.
- [ ] **Domain B**: Complete basic CLI wrapper, routing commands to internal handlers.
- [ ] **Sync Point**: Run race detection tests (`go test -race`) on raw I/O layers.

### Phase 3: Staging Integration (`add`)
- [ ] **Domain B**: Build the concurrent directory walker for `purr add .`.
- [ ] Integrate index updates with the worker pool for parallel zlib blob writing.
- [ ] **Sync Point**: Verify staging correctness when handling 10,000+ files.

### Phase 4: Commit Integration (`commit`)
- [ ] **Domain A & B**: Implement the full atomic `purr commit` workflow.
- [ ] Clear index post-commit and verify that tree entries point to correct content blobs.
- [ ] **Sync Point**: Confirm snapshot consistency matches Git storage architecture.

### Phase 5: Revert & Benchmark Polish
- [ ] **Domain B**: Implement parallel file restoration for `purr revert`.
- [ ] Conduct end-to-end stress tests with complex nested workspaces.
- [ ] Run benchmark comparisons (Sequential vs Parallel storage engine).
- [ ] **Sync Point**: No race conditions, and all unit tests pass with `100%` safety.

---

## Success Criteria & Risk Management

### Core Goals
- [x] Concurrent updates to `.purr/index` must be perfectly atomic.
- [x] Bulk operations (`purr add .`) must scale linearly with CPU core counts.
- [x] `.purr/` objects must remain fully readable and valid after extreme concurrency stress testing.
- [x] Zero external dependencies beyond the Go standard library.

### Risk Mitigation Strategy

| Identified Risk | Impact | Mitigation Plan |
| :--- | :---: | :--- |
| **Race Conditions** | High | Enforce continuous testing with `go test -race` on all build pipelines. |
| **Goroutine Leaks / Exhaustion** | High | Use strictly bounded worker pools using semaphore channels. |
| **Index Serialization Failure** | Med | Write updates to a temporary file (`index.tmp`), then perform atomic filesystem renames (`os.Rename`). |
| **File I/O Bottlenecks** | Med | Leverage memory-mapped pages (`syscall.Mmap`) for large object reading. |