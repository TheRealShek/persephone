# Go Git Clone - Phase 1: Technical Breakdown & Project Plan

## Architecture Overview

### On-Disk Structure
```
.git/
├── objects/              # Content-addressable storage (blobs, trees, commits)
│   ├── ab/
│   └── cd/
├── refs/
│   └── heads/
│       └── main          # Branch pointers
├── index                 # Staging area (serialized)
├── HEAD                  # Current branch reference
└── logs/
    └── HEAD              # Commit history log
```

### Core Data Structures

**Blob**: Represents file content
- Hash: SHA1(content)
- Stored at `.git/objects/XX/YYYYYY...`

**Tree**: Represents directory state
- Contains entries: `{name, mode, hash}`
- Hash: SHA1(serialized tree)

**Commit**: Represents snapshot
- Contains: `{tree_hash, parent_hash, author, message, timestamp}`
- Hash: SHA1(serialized commit)

**Index**: Staging area
- Maps filepath → `{file_hash, mode, timestamp}`
- Serialized format (simple binary or JSON for Phase 1)

---

## Task Breakdown by Role

### Developer A: Object Storage & Core Hashing

**Tasks (Priority Order)**
1. **Initialize repository structure** (`git init`)
   - Create `.git/` directory with subdirectories
   - Initialize HEAD, index, refs files
   - ~2 hours

2. **Implement concurrent blob storage & hashing**
   - SHA1 hashing of file content using worker goroutines (for bulk operations)
   - Write blobs to `.git/objects/XX/YYYYYY...` with concurrent file I/O
   - Read blobs by hash (concurrent reads with `sync.RWMutex` for cache)
   - Use channels for hash job distribution
   - ~4 hours

3. **Implement parallel tree generation**
   - Traverse file tree concurrently with goroutines (one goroutine per directory)
   - Use worker pool pattern to avoid goroutine explosion
   - Synchronize results with `sync.WaitGroup` and channels
   - Serialize/deserialize trees efficiently
   - Hash trees correctly
   - ~5 hours

4. **Build commit object model with concurrent access**
   - Create and serialize commits atomically with `sync.Mutex`
   - Write/read commits with goroutine-safe operations
   - Link commits via parent references
   - Implement commit log caching with concurrent-safe map
   - ~4 hours

**Subtotal: ~15 hours** (slightly more due to concurrency patterns, but faster execution)

---

### Developer B: Commands & Index Management

**Tasks (Priority Order)**
1. **CLI argument parsing**
   - Parse `git add <file>` and `git add .`
   - Parse `git commit -m "message"`
   - Parse `git revert <commit_hash>`
   - ~2 hours

2. **Implement concurrent `git add` command**
   - Parallel file scanning with goroutines (walk tree concurrently)
   - Hash files in worker pool (channels for job distribution)
   - Update index atomically with `sync.Mutex`
   - Handle `.` for all files efficiently
   - Handle individual file paths
   - Integrate with Developer A's blob storage
   - ~5 hours

3. **Implement `git commit` command**
   - Read index and create tree (reuse Dev A's concurrent tree builder)
   - Write commit object
   - Update branch pointer (HEAD) atomically
   - Clear index post-commit
   - ~4 hours

4. **Implement concurrent `git revert` command**
   - Lookup commit by hash
   - Restore files in parallel with goroutines
   - Create new commit atomically
   - Handle concurrent file writes safely
   - ~4 hours

5. **Add helpful output & error handling with logging**
   - Status messages for add/commit
   - Error messages for invalid operations
   - Concurrent-safe logging (buffered channels or `log` package)
   - ~2 hours

**Subtotal: ~17 hours** (concurrency adds complexity, but parallel operations deliver speed gains)

---

## Integration Points

- **Developer A → Developer B**: Concurrent blob/tree/commit APIs (goroutine-safe with mutexes/channels)
- **Dev B → Dev A**: Index format specification (agree on JSON or binary)
- **Concurrency contracts**: Define which operations are thread-safe, which require locking
- **Sync Point (Day 3)**: Test full `add → commit → revert` workflow under concurrent load
- **Dev A pre-work**: Finalize object storage format + concurrency guarantees before Dev B builds commands

---

## Technology Stack

- **Language**: Go 1.21+
- **Hashing**: `crypto/sha1`
- **File I/O**: `os`, `io/ioutil`, `filepath`
- **Serialization**: `encoding/json` (simple) or custom binary (optimized)
- **CLI**: `flag` package or `cobra` (if needed)
- **Concurrency**: `goroutines`, `channels`, `sync.WaitGroup`, `sync.Mutex`

---

## Project Management Checklist

### Implementation Phases

**Phase 1: Setup & Architecture**
- [ ] Create Go project repository
- [ ] Set up directory structure
- [ ] Define concurrency model (mutex vs channels vs atomic operations)
- [ ] Dev A: Start concurrent blob storage + define thread-safety guarantees
- [ ] Dev B: Outline CLI structure and argument parsing
- **Checkpoint**: Core project skeleton + concurrency architecture locked in

**Phase 2: Core Layer (Dev A) & CLI Layer (Dev B)**
- [ ] Dev A: Complete concurrent blob + parallel tree implementation
- [ ] Dev B: Complete argument parsing + basic CLI structure
- [ ] **Sync**: Review concurrent API surface together
- [ ] Agree on index serialization format + lock strategy
- [ ] Define channels for job distribution (Dev A ↔ Dev B coordination)
- **Checkpoint**: Object storage + CLI framework done; concurrency contracts documented

**Phase 3: Integration - Add Command**
- [ ] Dev A: Finish commit object model with atomic operations
- [ ] Dev B: Implement concurrent `git add` (parallelized file scanning + hashing)
- [ ] **Sync**: First end-to-end test under concurrent load (multiple add operations in parallel)
- **Checkpoint**: Basic `add` working with concurrent storage layer

**Phase 4: Integration - Commit Command**
- [ ] Dev B: Implement `git commit` command
- [ ] Dev A: Code review + concurrency bug fixes
- [ ] **Sync**: Test full `add → commit` workflow (concurrent adds followed by atomic commit)
- [ ] Verify `.git/` structure matches Git
- [ ] Stress test with large file trees
- **Checkpoint**: `add` and `commit` fully functional + concurrent-safe

**Phase 5: Revert & Polish**
- [ ] Dev B: Implement concurrent `git revert` command
- [ ] Dev A: Support helper functions for parallel file restoration
- [ ] **Sync**: Full integration test (concurrent adds → commit → parallel revert)
- [ ] Concurrency testing & race condition fixes (`go test -race`)
- [ ] Benchmark operations (compare single-threaded vs goroutine speedup)
- [ ] Write README with concurrency notes + usage examples
- **Checkpoint**: Phase 1 complete, tested for correctness + concurrency safety

### Success Criteria

- [ ] `git add <file>` updates index correctly (concurrent-safe)
- [ ] `git add .` stages all modified files in parallel
- [ ] `git commit -m "msg"` creates atomic commit with tree snapshot
- [ ] `git revert <hash>` restores files in parallel and creates revert commit
- [ ] `.git/` structure matches real Git layout
- [ ] No race conditions (`go test -race` passes)
- [ ] No external dependencies beyond stdlib
- [ ] All tests passing (unit + integration + concurrent)
- [ ] Benchmarks show measurable speedup on large trees vs sequential approach

### Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Race conditions | Use `go test -race` continuously; document mutex/channel usage |
| Goroutine explosion on large trees | Worker pool pattern with bounded goroutine count |
| Index corruption under concurrent adds | Atomic locking strategy; test concurrent add scenarios |
| Performance regression | Benchmark before/after; track goroutine overhead |
| Deadlocks | Single lock hierarchy; avoid nested locks; use channels for coordination |
| Serialization format mismatch | Agree on format Day 2 morning + lock it in |

---