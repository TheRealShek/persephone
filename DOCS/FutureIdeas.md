# Persephone — Phase 2 Opportunities & Roadmap

Persephone is designed to question legacy assumptions in version control. Following the core implementation of Phase 1 (basic object storage, staging, and commits), Phase 2 will focus on rich history inspection, parallel branch/state mutation, language-aware semantic merges, and true decentralized coordination.

---

## 1. Inspection Engine

To transition from a simple snapshot tool to an active development workspace, Persephone needs powerful inspection capabilities.

### `purr log` (History Visualizer)
- **Concept**: Traverse the Commit Directed Acyclic Graph (DAG) concurrently.
- **Design & Architecture**:
  - Rather than sequentially walking parent pointers, implement a concurrent graph traversal engine using a worker pool.
  - Cache commit nodes in Pebble/Badger DB with pre-indexed metadata (e.g., author, date, and custom fields).
  - Implement a terminal user interface (TUI) via `bubbletea` to allow developers to interactively fold/unfold branch merges and inspect commits on the fly.

### `purr status` & `purr diff` (Fast Workspace Diffing)
- **Concept**: Compare the current working tree, index, and `HEAD` commit.
- **Design & Architecture**:
  - **Concurrent Diffing**: Split file comparisons across goroutines. If the user changes 50 files, diff them in parallel using an efficient Myers diff implementation written in pure Go.
  - **In-Memory Cache**: Use a persistent background daemon (similar to `fsnotify`) to listen to file system change notifications. `purr status` should return instantly (O(1) complexity) instead of scanning the full working directory.

---

## 2. Branch & State Management

Moving beyond linear commit chains requires robust, concurrent branching primitives.

### `purr branch` & `purr checkout`
- **Concept**: Create, list, delete, and switch between branches.
- **Design & Architecture**:
  - **Lightweight Refs**: Branches are simply files under `.purr/refs/heads/` containing a commit hash. Creating a branch is a fast O(1) write.
  - **Parallel Checkouts**: Switching branches (`checkout`) involves updating the working directory to match the target commit's tree. We will parallelize file creation, staging, and directory cleanups, using memory-mapped file buffers for maximum throughput.
  - **Safe Stashing**: Implement a concurrent-safe stash system (`purr stash`) that serializes working tree changes into temporary tree objects in the Pebble/Badger database.

---

## 3. Semantic Merge Engine

Legacy Git merges line-by-line, leading to superficial conflicts when functions are reordered or formatted. Persephone will solve this at the semantic level.

### `purr merge` (Language-Aware Merging)
- **Concept**: Ast-based three-way merging.
- **Design & Architecture**:
  - **AST Parsing**: If a merge conflict is detected, the merge engine parses the base, local, and remote files into Abstract Syntax Trees (ASTs) using Go's extensive parsing ecosystem (e.g., `go/parser` for Go, tree-sitter bindings for other languages).
  - **Semantic Resolution**: If Developer A added a parameter to a function at the top of the file, and Developer B moved the function to the bottom of the file without changing its signature, legacy Git fails. Persephone's AST merge engine resolves this automatically.
  - **Visual Merge Conflict Interface**: Introduce a Bubble Tea TUI that allows developers to step through unresolved AST node conflicts side-by-side.

---

## 4. Peer-to-Peer Synchronization

True decentralization means operating without needing central registries like GitHub or GitLab.

### `purr push` & `purr pull` (Distributed Coordination)
- **Concept**: Direct node-to-node object transfers over libp2p.
- **Design & Architecture**:
  - **DHT Discovery**: Use a Kademlia Distributed Hash Table (DHT) for peer discovery. When two developers are on the same local network, they can auto-discover each other and sync branches instantly.
  - **Vector Clocks & DAG Syncing**: Use vector clocks to trace concurrent edits in a true distributed masterless system. Syncing is treated as a DAG reconciliation problem, solved using peer-to-peer set-reconciliation protocols.
  - **End-to-End Encryption**: Allow optional repository encryption where objects are encrypted with Ed25519-derived keys before being transmitted to other peers.