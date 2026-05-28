# Git Commands: Deep Dive into Internal Workings

This guide details the internal mechanisms of core Git commands. Understanding these low-level operations provides a solid foundation for designing and optimizing custom version control systems like **Persephone**.

---

## Git's Architectural Pillars

At its core, Git orchestrates changes across four primary conceptual environments:

```
[ Working Directory ]  <--- (Your actual files on disk)
       │
       ▼  (git add)
[ Staging Area (Index) ]  <--- (Binary metadata tracking staging state)
       │
       ▼  (git commit)
[ Git Object Database ] <--- (Content-addressable storage: Blobs, Trees, Commits)
       ▲
       │
[ Refs & Head Pointers ] <--- (Lightweight files pointing to commits)
```

---

## Command-by-Command Internals

### 1. `git init`
Initializes an empty repository, creating the necessary structure to track project snapshots.

* **Internal Steps**:
  1. Creates the hidden `.git` metadata directory at the repository root.
  2. Generates standard internal subdirectories:
     - `objects/`: The content-addressable storage.
     - `refs/heads/`: Storage for local branch pointers.
     - `hooks/`: Shell script templates for lifecycle events.
  3. Creates the standard `config` file containing local options.
  4. Creates the `HEAD` file, setting it to `ref: refs/heads/main` (pointing to the default branch).
* **Data Structures Affected**: Adds a clean `.git` folder structure to the local file system.

> [!NOTE]
> No database objects are generated during `git init`. The object store (`.git/objects/`) remains completely empty until the first file is staged or committed.

---

### 2. `git clone`
Downloads a remote repository's historical snapshots and extracts the latest revision into the local workspace.

* **Internal Steps**:
  1. Creates a target directory and initializes it by invoking the equivalent of `git init`.
  2. Resolves connection to the remote server via the specified protocol (HTTPS, SSH, or Git).
  3. Downloads all historical objects (commits, trees, blobs) and stores them in `.git/objects` (often packed as a single compressed `.pack` file for network efficiency).
  4. Creates tracking references for remote branches under `refs/remotes/origin/*`.
  5. Inspects `HEAD` of the remote, points local `HEAD` to the corresponding local branch tracking it, and checks out that snapshot into the working directory.
* **Data Structures Affected**: Populates the local object store and ref pointers with the entire history of the project.

> [!TIP]
> Cloning is simply a network fetch followed by a full directory checkout. Git downloads the entire DAG first, then extracts the specific target commit's files to your disk.

---

### 3. `git add`
Stages modifications from the working directory, preparing them to be committed.

* **Internal Steps**:
  1. Recursively scans the specified directories and files.
  2. Computes the SHA-1 hash for each file's raw content.
  3. Creates a **blob** object (containing the compressed content) and writes it to `.git/objects/XX/YYYY...` (where `XX` is the first two characters of the hash, and `YYYY...` is the remaining 38 characters).
  4. Updates the binary index file (`.git/index`) to map the file's path to its new blob SHA-1 hash, permissions, file size, and timestamp data.
* **Data Structures Affected**:
  - **Blobs**: Written directly to the object store.
  - **Index File**: Staging metadata is updated.

> [!IMPORTANT]
> `git add` does not affect branch history. It is the command that actually writes file content to disk. If you make further modifications to a staged file, those changes will not be included in the commit unless you run `git add` again.

---

### 4. `git commit`
Takes a snapshot of all currently staged changes and records it permanently in the repository history.

* **Internal Steps**:
  1. Reads the current binary staging index (`.git/index`).
  2. Generates a hierarchical tree structure representing the staged directory state. It creates a **tree** object for every directory and subdirectory, containing entries pointing to child trees or blobs.
  3. Writes all new tree objects to the object database.
  4. Creates a **commit** object containing:
     - The SHA-1 hash of the root tree object representing the snapshot.
     - The SHA-1 hash of the parent commit(s) (if any).
     - Author and Committer signatures (name, email, timestamp).
     - The commit message.
  5. Writes the compressed commit object to `.git/objects`.
  6. Updates the current branch ref file (e.g., `.git/refs/heads/main`) to point to the new commit's SHA-1.
* **Data Structures Affected**: Writes new Tree and Commit objects, and updates branch refs.

> [!WARNING]
> Git commits are immutable. Because a commit hash is computed from its tree, author, timestamp, parent, and message, modifying *any* detail in history generates an entirely new commit hash, breaking all downstream references.

---

### 5. `git push`
Transfers local commits and updates remote branch pointers on the remote repository.

* **Internal Steps**:
  1. Performs a handshake with the remote server, exchanging a list of commit hashes to determine what history the remote is missing.
  2. Compiles a compressed "packfile" containing only the missing objects (commits, trees, and blobs).
  3. Sends the package securely over the network (HTTPS or SSH).
  4. Instructs the remote server to update its branch reference (e.g., `refs/heads/main`) to match the sender's local branch commit hash.
* **Data Structures Affected**: The remote's database gains the compressed packfile, and its branch references are updated.

---

### 6. `git pull`
Fetches the latest remote changes and immediately merges them into the current active branch.

* **Internal Steps**:
  1. **Fetch Phase**: Queries the remote, downloads any new commit, tree, and blob objects into the local `.git/objects` folder, and updates the local remote-tracking branch (e.g., `refs/remotes/origin/main`).
  2. **Merge/Rebase Phase**: Integrates the downloaded commits:
     - **Merge (Default)**: Creates a new "merge commit" with two parents—the local branch commit and the remote tracking branch commit—and updates the working directory.
     - **Rebase**: Temporarily stashes local modifications, fast-forwards to the remote's latest commit, and replays local commits sequentially on top.
* **Data Structures Affected**: Populates the local object store and updates the local branch pointers and working directory.

---

## The Git Object Model

Git represents all files, directories, and history using four simple, immutable, content-addressable objects:

| Object Type | Role | Content | Key Identifier |
| :--- | :--- | :--- | :--- |
| **Blob** | File Content | Raw compressed data (no file names or metadata) | SHA-1 of content |
| **Tree** | Directories | List of entries: `[mode, type, hash, name]` | SHA-1 of list |
| **Commit** | Snapshots | Pointer to root Tree, Parent commit(s), Author, Message | SHA-1 of metadata |
| **Ref** | Label / Pointer | A text file containing a single commit hash | File path on disk |

---

## Why Internal Workings Matter

Deep understanding of Git internals allows you to:
1. **Safely Rewrite History**: Perform precise rebasing or interactive history cleanups confidently.
2. **Optimize Monorepo Speed**: Understand how indexing and stat caching work, allowing for custom tooling optimizations.
3. **Recover "Lost" Code**: Find dangling commits using `git reflog` or direct scans of the `.git/objects` directory, even if they were checked out or deleted from active branches.