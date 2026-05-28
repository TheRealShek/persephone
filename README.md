# Persephone — Reimagining Git (in Go)

> **Note:** Personal experimental project. Originally created in collaboration with [Chandranil Bakshi](https://github.com/chandranilbakshi). Since they are no longer working on it, I am continuing the project here. The original repository is available at [chandranilbakshi/persephone](https://github.com/chandranilbakshi/persephone).
> No PRs. No contributions. Don’t ask.

## Vision (why this exists)
Git is legendary.
It’s also old. The world changed: SSDs, CI, huge monorepos, concurrency. Git still behaves like a 2005 CLI tool.

Persephone = “what if we rebuilt Git 2025-first?”

- concurrency first
- modern storage backends
- modern UX + tooling
- experimentation without sacred cows

## Currently Implemented
| Command | Status |
|---|---|
| `purr init` | works |
| `purr add` | works (concurrent hashing, skip unchanged) |
| `purr ls-files` | works |
| `purr config` | works |
| `purr commit` | works (SHA-1, Git-like index) |

Everything else (branch, merge, remote, log, diff, etc.) → **not implemented yet**.

See: [`Purr Commands Guide`](./Purr%20Commands%20Guide)

## Future Directions
(no guarantees — this is a lab)

- blobs/trees/commits in Badger/Pebble instead of flat `.git/objects`
- JSON/ProtoBuf commit metadata
- semantic merge engine (AST-based)
- plugin interface (Go interfaces)
- real distributed sync (IPFS/libp2p)
- TUI + visualizations
- optional encryption + Ed25519 signing

## Status
This is a prototype.
It’s here to learn, to question assumptions, and to invent. Not to be “production ready”.

If you want a stable DVCS: use Git.
If you want to explore what the *next* DVCS could look like: that’s why Persephone exists.
