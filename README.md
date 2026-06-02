# Persephone (purr)
> *What if Git was reborn in Go, loved concurrency, and faster than classic Git?*

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](#)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue)](#)
[![Status](https://img.shields.io/badge/status-experimental-orange)](#)

---

## About the Project

**Git is legendary, but it's also aging.**
The software landscape has evolved dramatically with the advent of SSDs, CI/CD pipelines, massive monorepos, and highly concurrent workloads. Yet, Git still behaves like a CLI tool from 2005.

Persephone (CLI tool `purr`) is an experimental lab exploring a simple question: *"What if we rebuilt Git with a 2025-first mindset?"*

**Core Tenets:**
- **Concurrency First:** Designed from the ground up to leverage Go's goroutines for blazing-fast operations (e.g., parallel file hashing).
- **Modern Storage:** Exploring modern storage backends beyond flat-file layouts.
- **Beautiful UX:** Semantic CLI output via lipgloss.
- **Extensible Design:** Built around content-addressed objects and Go packages that can evolve toward richer metadata and extensions.

---

## Demo

![Persephone CLI Demo](assets/demo.gif)

---

## Currently Implemented

The foundation of the VCS is being laid down. Here is the current command support:

| Command | Description | Status / Features |
|---|---|---|
| `purr init` | Initializes a new repository | Works (confirmation required before reinitialization) |
| `purr config` | Get and set global options | Works (Global JSON config) |
| `purr add` | Stages files into the index | Works (Concurrent hashing, skip unchanged) |
| `purr remove` | Removes tracked files | Works (Removes from index and working directory) |
| `purr ls` | Shows staged files | Works (formatted table, short hashes) |
| `purr commit` | Records changes | Works (Git-style commit objects, SHA-1) |
| `purr log` | Shows commit history | Works (HEAD ancestry, newest-to-oldest) |

> *Note: Everything else (branch, merge, remote, diff, etc.) is currently **not implemented**.*

---

## Future Directions

*(No guarantees — this is a lab!)*

### Near-term
- **Modern Metadata:** Explore optional structured metadata beyond the current Git-style commit payload.
- **Extensibility:** A robust plugin interface via Go interfaces.
- **Visualizations:** Extend `purr log` with a scrollable graph TUI.

### Long-term
- **Alternative Storage:** Storing blobs/trees/commits in Badger/Pebble instead of a flat `.purr/objects` structure.
- **Semantic Merging:** AST-based merge engine to understand actual code structure.
- **Distributed Sync:** Real peer-to-peer sync using IPFS/libp2p.
- **Security:** Optional encryption and Ed25519 commit signing out-of-the-box.

---

## ⚠️ Status & Disclaimer

**This is a prototype.**
It is built to learn, invent, and question assumptions. It is **not** production-ready.

> **Disclaimer:** This is a personal experimental project, originally created in collaboration with [Chandranil Bakshi](https://github.com/chandranilbakshi) and now continued here.
>
> **No PRs. No contributions. Don’t ask.**

If you want a stable, battle-tested DVCS: **use Git**.
If you want to explore what the *next* DVCS could look like: **explore Persephone**.
