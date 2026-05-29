# Persephone (purr)
> *A concurrent, experimental reimagining of Git in Go.*

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
- **Premium UX:** Gorgeous, semantic CLI outputs utilizing `lipgloss`.
- **Sacred-Cow-Free:** An uninhibited sandbox to test wild ideas and break assumptions.

> **Disclaimer:** This is a personal experimental project. Originally created in collaboration with [Chandranil Bakshi](https://github.com/chandranilbakshi). Since they are no longer working on it, I am continuing the project here. The original repository is available at [chandranilbakshi/persephone](https://github.com/chandranilbakshi/persephone).
> 
> **No PRs. No contributions. Don’t ask.**

---

## Currently Implemented

The foundation of the VCS is being laid down. Here is the current command support:

| Command | Description | Status / Features |
|---|---|---|
| `purr init` | Initializes a new repository | Works (Sets up `.purr` database) |
| `purr config` | Get and set global options | Works (Global JSON config) |
| `purr add` | Stages files into the index | Works (Concurrent hashing, skip unchanged) |
| `purr ls` | Shows staged files | Works (Premium semantic output) |
| `purr commit` | Records changes | Works (JSON commit objects, SHA-1) |

> *Note: Everything else (branch, merge, remote, log, diff, etc.) is currently **not implemented**.*

---

## Future Directions

*(No guarantees — this is a lab!)*

- **Alternative Storage:** Storing blobs/trees/commits in Badger/Pebble instead of a flat `.purr/objects` structure.
- **Modern Metadata:** Using JSON or ProtoBuf for commit metadata.
- **Semantic Merging:** AST-based merge engine to understand actual code structure.
- **Extensibility:** A robust plugin interface via Go interfaces.
- **Distributed Sync:** Real peer-to-peer sync using IPFS/libp2p.
- **Visualizations:** Advanced TUI and visual graph representations.
- **Security:** Optional encryption and Ed25519 commit signing out-of-the-box.

---

## Status

**This is a prototype.**
It is built to learn, invent, and question assumptions. It is **not** production-ready.

- If you want a stable, battle-tested DVCS: **use Git**.
- If you want to explore what the *next* DVCS could look like: **explore Persephone**.
