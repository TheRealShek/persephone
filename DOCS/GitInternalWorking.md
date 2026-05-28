# Git Commands: Internal Workings Guide

## Understanding Git's Foundation

Git stores all repository data in a hidden `.git` directory. The core of Git revolves around four main concepts: the working directory (your actual files), the staging area (files marked for commit), the Git object database (where commits, trees, and blobs live), and refs (pointers to commits).

---

## git init

**Purpose**: Initialize a new Git repository in a directory.

**Internal Processes**:
1. Git creates a `.git` directory containing subdirectories like `objects`, `refs`, `hooks`, and `info`
2. Creates initial configuration files (`config`, `HEAD`) 
3. Initializes `HEAD` to point to `refs/heads/main` (the default branch)
4. Creates an empty object database ready to store commits

**Data Structures Affected**: The filesystem gains a `.git` folder that serves as the repository's backbone.

---

## git clone

**Purpose**: Copy a remote repository to your local machine.

**Internal Processes**:
1. Git creates a new directory and initializes it with `git init`
2. Contacts the remote server using the specified protocol (HTTPS, SSH, or Git protocol)
3. Fetches all objects from the remote repository (commits, trees, blobs) and stores them in `.git/objects`
4. Creates `refs/remotes/origin/*` to track remote branches
5. Checks out the default branch into your working directory using the commit tree structure

**Data Structures Affected**: Populates the object database with all historical commits and creates remote tracking branches.

**Key Insight**: Cloning is essentially a fetch followed by a checkout—Git downloads everything, then extracts the files you need to work with.

---

## git add

**Purpose**: Move changes from the working directory to the staging area (index).

**Internal Processes**:
1. Git scans the files you specify and computes their SHA-1 hashes
2. For each file, Git creates a blob object (containing the file's content) and stores it in `.git/objects`
3. Updates the index file (`.git/index`) with references to these blobs and file metadata (permissions, timestamps)
4. The index is a binary file that tracks what will be included in the next commit

**Data Structures Affected**: 
- Blob objects are created and added to the object database
- The index file is updated with staging information

**Key Insight**: `git add` doesn't commit anything—it just prepares objects and updates the staging index. If you modify a file after staging it, the new changes aren't staged until you `git add` again.

---

## git commit

**Purpose**: Create a snapshot of staged changes as a permanent record.

**Internal Processes**:
1. Git takes the current state of the index (staging area) and creates a tree object
2. This tree object is a hierarchical structure representing your project's directory layout, with references to blob objects (files) and subtree objects (subdirectories)
3. Git creates a commit object containing:
   - A reference to the tree object
   - References to parent commit(s) 
   - Metadata (author, committer, timestamp, commit message)
   - A SHA-1 hash computed from all this data
4. Updates the current branch pointer (in `refs/heads/[branch-name]`) to point to this new commit object
5. Updates `HEAD` if on a branch, or stays detached if you're on a specific commit

**Data Structures Affected**:
- Tree objects (representing directory structure)
- Blob objects (file content)
- Commit objects (the snapshots themselves)
- Branch refs (updated to point to the new commit)

**Key Insight**: Commits are immutable. Every commit's hash is computed from its content, parent, and metadata. Changing anything in history creates a new hash, which is why rewriting history is dangerous in shared repositories.

---

## git push

**Purpose**: Upload your local commits to a remote repository.

**Internal Processes**:
1. Git compares your local branch against the remote branch to identify commits you have that the remote doesn't
2. Transfers only the new objects (commits, trees, blobs) to the remote server using a network protocol
3. Updates the remote branch pointer to match your local branch pointer
4. The remote repository's object database receives all necessary objects to reconstruct your commits

**Network Protocol Details**: 
- HTTPS: Git packages objects in a compressed format and sends them via HTTP POST requests
- SSH: Uses SSH tunneling to send the same object data securely
- Git Protocol: A lightweight protocol specifically designed for Git

**Data Structures Affected**: Remote branch refs are updated; the remote's object database grows with your new commits.

**Key Insight**: Push only transfers new commits—Git's efficiency comes from storing immutable objects once and reusing them. If multiple people commit the same file content, Git stores it once as a single blob.

---

## git pull

**Purpose**: Fetch remote changes and integrate them into your local branch.

**Internal Processes**:
1. **Fetch phase**: Git contacts the remote server and downloads any new commits, trees, and blobs from the remote branch into your local `.git/objects` directory
2. Updates your remote tracking branch (`refs/remotes/origin/[branch]`) to point to the remote's latest commit
3. **Merge/Rebase phase** (default is merge):
   - If doing a merge: Git creates a new commit object that has two parents (your current branch and the remote branch), incorporating all changes from both
   - If rebasing: Git replays your local commits on top of the remote's latest commit
4. Updates your working directory with the merged/rebased result

**Data Structures Affected**: Object database receives new commits from the remote; branch pointers and possibly the working directory are updated.

**Key Insight**: `git pull` is convenience command combining `git fetch` and `git merge`. Understanding it as two separate operations helps you troubleshoot integration issues and gives you more control over how changes are combined.

---

## How These Commands Interact: A Workflow Example

When you work in Git, you're moving data through layers:

1. You modify files in your working directory
2. `git add` stages them by creating blobs and updating the index
3. `git commit` transforms the index into a tree structure and creates an immutable commit object
4. `git push` transfers your commits to the remote server
5. A collaborator does `git pull`, which fetches your commits and updates their working directory

At each stage, Git is building or traversing its object graph—a directed acyclic graph (DAG) where commits point to their parents and trees point to their contents.

---

## The Git Object Model Simplified

- **Blobs**: Raw file content, identified by SHA-1 hash of their data
- **Trees**: Collections of blobs and other trees, representing directory structure
- **Commits**: Snapshots containing a tree, parent commit(s), metadata, and a unique hash
- **Refs**: Simple pointers to commits (branches, tags, HEAD)

Everything is content-addressable by SHA-1 hash, meaning Git's integrity is built into its structure. You can't modify history without Git detecting it.

---

## Why This Matters

Understanding these internals helps you:
- Recover lost commits (they're still in `.git/objects`)
- Understand why certain operations are fast (Git only transfers what's needed)
- Debug merge conflicts (you can see the tree structure)
- Know when it's safe to rewrite history (only on private branches)
- Use advanced features like rebasing, cherry-picking, and refspecs confidently