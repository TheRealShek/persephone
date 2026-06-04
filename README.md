# Persephone (purr)

> _What if Git was reborn in Go, loved concurrency, and faster than classic Git?_

[![Build Status](https://github.com/TheRealShek/persephone/actions/workflows/release.yml/badge.svg)](https://github.com/TheRealShek/persephone/actions/workflows/release.yml)
[![Release](https://img.shields.io/github/v/release/TheRealShek/persephone)](https://github.com/TheRealShek/persephone/releases/latest)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org/doc/go1.25)
[![Go Report Card](https://goreportcard.com/badge/github.com/TheRealShek/persephone)](https://goreportcard.com/report/github.com/TheRealShek/persephone)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Demo

![Persephone CLI Demo](assets/demo.gif)

---

## About the Project

Persephone (CLI tool `purr`) is an experimental lab exploring a simple question: _"What if we rebuilt Git with a 2025-first mindset?"_ It reimagines version control for modern hardware, massive repositories, and concurrent workloads.

**Core Tenets:**

- **Concurrency First:** Designed from the ground up to leverage Go's goroutines for blazing-fast operations (e.g., parallel file hashing).
- **Modern Storage:** Exploring modern storage backends beyond flat-file layouts.
- **Beautiful UX:** Semantic CLI output via lipgloss.
- **Extensible Design:** Built around content-addressed objects and Go packages that can evolve toward richer metadata and extensions.

---

## Benchmarks ⚡

We built Persephone with concurrency in mind. But just how much faster is it?

Check out the **[TheRealShek/persephone-bench](https://github.com/TheRealShek/persephone-bench)** repository for detailed performance benchmarks comparing `purr` directly against classic `git`.

---

## Installation

### Option 1: Install Script (Linux)

You can easily install the latest release on Linux (amd64/arm64) using the provided installation script:

```bash
curl -fsSL https://raw.githubusercontent.com/TheRealShek/persephone/main/install.sh | sh
```

### Option 2: Manual (Linux & macOS)

If you are on macOS or prefer to install manually, download the pre-compiled binary directly from the [Releases](https://github.com/TheRealShek/persephone/releases) page.

**1. Choose the correct download for your system:**

- **Mac with Intel chip:** `persephone_darwin_amd64.tar.gz`
- **Mac with Apple Silicon (M1/M2/M3):** `persephone_darwin_arm64.tar.gz`
- **Linux (64-bit PC):** `persephone_linux_amd64.tar.gz`
- **Linux (ARM/Raspberry Pi):** `persephone_linux_arm64.tar.gz`

**2. Extract and install:**
Once downloaded, extract the archive and move the `purr` binary to a folder in your `$PATH` (like `/usr/local/bin/`).

```bash
# 1. Download the archive (example for Linux AMD64)
curl -LO https://github.com/TheRealShek/persephone/releases/latest/download/persephone_linux_amd64.tar.gz

# 2. Extract it
tar -xzf persephone_linux_amd64.tar.gz

# 3. Move the binary into your PATH
sudo mv purr /usr/local/bin/

# 4. Verify installation
purr --version
```

---

## Currently Implemented

The foundation of the VCS is being laid down. Here is the current command support:

| Command       | Description                  | Status / Features                                     |
| ------------- | ---------------------------- | ----------------------------------------------------- |
| `purr init`   | Initializes a new repository | Works (confirmation required before reinitialization) |
| `purr config` | Get and set global options   | Works (Global JSON config)                            |
| `purr add`    | Stages files into the index  | Works (Concurrent hashing, skip unchanged)            |
| `purr remove` | Removes tracked files        | Works (Removes from index and working directory)      |
| `purr ls`     | Shows staged files           | Works (formatted table, short hashes)                 |
| `purr commit` | Records changes              | Works (Git-style commit objects, SHA-1)               |
| `purr log`    | Shows commit history         | Works (HEAD ancestry, newest-to-oldest)               |

> _Note: Everything else (branch, merge, remote, diff, etc.) is currently **not implemented**._

---

## Future Directions

_(No guarantees — this is a lab!)_

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
If you want to explore what the _next_ DVCS could look like: **explore Persephone**.
