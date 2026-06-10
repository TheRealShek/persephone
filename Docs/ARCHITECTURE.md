# Purr — Architecture Design & Sequence Flows

This document details the software design, sequence flows, and internal implementations of each custom **Purr** command.

---

## 1. Package Structure

```
persephone/
├── cmd/                             # Cobra command definitions (porcelain)
│   ├── purr/
│   │   └── main.go                  # CLI binary entry point
│   ├── add.go                       # Cobra 'add' command definition
│   ├── commit.go                    # Cobra 'commit' command definition
│   ├── config.go                    # Cobra 'config' command definition
│   ├── init.go                      # Cobra 'init' command definition
│   ├── log.go                       # Cobra 'log' command definition
│   ├── ls.go                        # Cobra 'ls' command definition
│   ├── remove.go                    # Cobra 'remove' command definition
│   └── root.go                      # Cobra root command definition
│
├── internal/
│   ├── config/                      # Configuration management
│   │   ├── config.go                # Read/write user configuration
│   │   └── types.go                 # Configuration types
│   │
│   ├── fsutil/                      # Filesystem utilities
│   │   └── fsutil.go                # File existence and traversal walking
│   │
│   ├── hash/                        # Hashing and compression
│   │   └── shaFunctions.go          # Blob and tree hashing, zlib writes
│   │
│   ├── index/                       # Staging area index management
│   │   ├── index.go                 # Binary index codec (DIRC reader/writer)
│   │   ├── types.go                 # Staging index structures
│   │   └── utils.go                 # Index population helpers
│   │
│   ├── objects/                     # Git-compatible objects representation
│   │   ├── commitFunctions.go       # Commit & Tree serialization and verification
│   │   ├── store.go                 # Content-addressed storage (zlib compression)
│   │   └── types.go                 # Object structures
│   │
│   ├── platform/                    # Platform-specific OS attributes and stat extraction
│   │   ├── hidden_unix.go
│   │   ├── hidden_windows.go
│   │   ├── stat.go
│   │   ├── stat_darwin.go
│   │   ├── stat_linux.go
│   │   └── stat_windows.go
│   │
│   ├── purrcommands/                # CLI Command implementation logic (plumbing)
│   │   ├── add.go
│   │   ├── commit.go
│   │   ├── config.go
│   │   ├── init.go
│   │   ├── log.go
│   │   ├── ls.go
│   │   └── remove.go
│   │
│   ├── refs/                        # Reference management
│   │   └── refs.go                  # HEAD resolution and branch updates
│   │
│   ├── repository/                  # Repository handle
│   │   └── repository.go            # Path resolution and validation
│   │
│   ├── testutils/                   # Test configuration helpers
│   │   └── helpers.go
│   │
│   └── ui/                          # Styled terminal rendering components
│       ├── components.go
│       └── styles.go
│
├── Docs/
├── Makefile
├── go.mod
└── README.md
```

---

## 2. Package Responsibilities

| Package | Owns | Imports |
|---|---|---|
| `cmd/` | Cobra command definitions, flag parsing, output formatting, exit handlers | `internal/purrcommands`, `internal/objects`, `internal/ui` |
| `internal/config` | Global configuration loading/writing (`~/.purrconfig`) | Standard library |
| `internal/fsutil` | File existence checks and workspace directory crawling | Standard library |
| `internal/hash` | Content-addressed SHA-1 hashing, blob/tree serialization and zlib storage orchestration | `internal/objects` |
| `internal/index` | Staging area catalog codec, stat cache checking, binary DIRC parser | `internal/platform` |
| `internal/objects` | Git-compatible VCS object builders (Blob, Tree, Commit), commit verification and loose store I/O | `internal/config` |
| `internal/platform` | Low-level OS-specific stat attributes mapping and hidden attributes management via build tags | Standard library |
| `internal/purrcommands` | Core execution flow engine for every VCS subcommand (add, commit, config, init, log, ls, remove) | `internal/config`, `internal/index`, `internal/objects`, `internal/refs`, `internal/fsutil`, `internal/hash`, `internal/ui` |
| `internal/refs` | Reference pointer storage (HEAD reading/writing, symbolic branch reference updates) | Standard library |
| `internal/repository` | Central repository handle, repository folder paths mapping and verification | Standard library |
| `internal/ui` | Lipgloss-based terminal styles and components for user output | `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles` |

---

## 3. Dependency Rules

```mermaid
graph TD
    CMD["cmd/"] --> PURR["internal/purrcommands/"]
    CMD --> REPO["internal/repository/"]
    CMD --> UI["internal/ui/"]
    PURR --> FS["internal/fsutil/"]
    PURR --> HASH["internal/hash/"]
    PURR --> INDEX["internal/index/"]
    PURR --> OBJ["internal/objects/"]
    PURR --> REFS["internal/refs/"]
    PURR --> CFG["internal/config/"]
    PURR --> UI
    HASH --> OBJ
    INDEX --> PLAT["internal/platform/"]
```

**Hard rules:**

1. **Nothing imports `cmd/`** — it's the outermost layer.
2. **`internal/ui/` is imported only for terminal representation** and styling.
3. **`internal/platform/` is selected by build tags** and only imported by packages requiring low-level OS attributes.
4. **All files and folders use lowercase module imports `persephone/internal/...`**

---

## 4. Command sequence flows

### 4.1 `purr init`

Initializes a local repository with the necessary directory hierarchy and metadata configuration.

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/init.go (Cobra)
    participant Core as internal/purrcommands/init.go
    participant OS as Filesystem

    User->>CLI: Runs "purr init"
    CLI->>Core: InitPurrDirectories(".")

    activate Core
    Core->>OS: Check whether ".purr" already exists
    alt Metadata root already exists
        Core-->>CLI: Return "repository already initialized"
        CLI-->>User: Ask whether to reinitialize [y/N]
        alt User confirms
            CLI->>Core: ReinitializePurrDirectories(".")
            Core->>OS: Restore missing scaffolding without overwriting metadata
            Core-->>CLI: Returns success status (nil)
            CLI-->>User: Prints "Reinitialized existing repository"
        else User declines or input ends
            CLI-->>User: Prints "Reinitialization cancelled"
        end
    else New repository
    Core->>OS: os.MkdirAll(".purr/{objects,refs/heads,logs}")
    Note over Core,OS: If OS is Windows, sets .purr directory as hidden

    Core->>OS: Write valid 12-byte header to ".purr/index"
    Note over Core,OS: Header: "DIRC" (4B) | Version 2 (4B) | Count 0 (4B)

    Core->>OS: Write "ref: refs/heads/main\n" to ".purr/HEAD"

    Core-->>CLI: Returns success status (nil)
    deactivate Core

    CLI-->>User: Prints "Initialized empty repository"
    end
```

1. **Invocation**: The user executes `purr init`. The runtime invokes the entrypoint in `cmd/init.go`.
2. **Directory Bootstrapping**: Core calls `InitPurrDirectories(".")` inside `internal/purrcommands/init.go`. It builds `.purr/objects`, `.purr/refs/heads`, and `.purr/logs`.
3. **Explicit Reinitialization Guard**: If `.purr` already exists, initial setup stops before touching metadata and the CLI asks for confirmation. An accepted reinitialization restores missing scaffolding while preserving index, HEAD, refs, and objects.
4. **OS-Specific Adjustments**: On Windows platforms, `.purr` is set to "hidden" using syscalls.
5. **Staging Index Creation**: Writes a valid 12-byte binary index header:
   - Magic signature: `"DIRC"` (4 bytes)
   - Staging Version: `2` (4 bytes, big-endian)
   - Initial count of entries: `0` (4 bytes, big-endian)
6. **HEAD Initialization**: Writes `"ref: refs/heads/main\n"` to `.purr/HEAD`, binding active tracking to the `main` branch.

### 4.2 `purr ls`

Lists all files currently tracked in the staging index.

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/ls.go (Cobra)
    participant Core as internal/purrcommands/ls.go
    participant Index as internal/index

    User->>CLI: Runs "purr ls [--debug]"
    CLI->>Core: ListFiles(showDebug)

    activate Core
    Core->>Index: ReadIndex(".purr/index")
    Index-->>Core: Slice of Index entries

    alt Index is Empty
        Core-->>CLI: Print "No files in index"
    else Index Contains Entries
        loop for each entry
            alt showDebug == true
                Core-->>CLI: Print detailed structural metadata
            else showDebug == false
                Core-->>CLI: Print simple "SHA-1  mode  path"
            end
        end
    end

    Core-->>CLI: Returns nil (success)
    deactivate Core
    CLI-->>User: Displays list outputs
```

1. **Loading Index**: The CLI calls `ListFiles(showDebug)` in `internal/purrcommands/ls.go`. It reads the binary database under `.purr/index` using the `index.ReadIndex` library helper.
2. **Empty Bounds Handling**: If the index contains `0` records, the command exits with `"No files in index"`.
3. **Output Rendering**:
   - **Default Mode**: Displays the calculated object hash, file mode, and relative path.
   - **Debug Mode**: Prints detailed binary index records, including timestamps (`mtime`, `ctime`), host attributes (`dev`, `ino`, `uid`, `gid`), file sizes, and stage parameters.

### 4.3 `purr config`

Manages configuration files on the local machine.

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/config.go (Cobra)
    participant Core as internal/purrcommands/config.go
    participant OS as ~/.purrconfig

    alt Read Key
        User->>CLI: Runs "purr config <key>"
        CLI->>Core: ConfigCommand(key)
        Core->>OS: Load config parameters
        OS-->>Core: Values
        Core-->>User: Prints value of key (e.g. user.name)
    else Write Key-Value
        User->>CLI: Runs "purr config <key> <value>"
        CLI->>Core: ConfigCommand(key, value)
        Core->>OS: Write updated key-value parameter
        OS-->>Core: Success
        Core-->>User: Prints confirmation message
    end
```

1. **Invocation**: The user executes `purr config <key> [value]`.
2. **CLI Routing**: Handles read or write modes depending on the argument length:
   - **Read Mode** (1 argument): Invokes `config.ReadConfig()` to load the global configuration file (`~/.purrconfig`) and outputs the value of the requested key.
   - **Write Mode** (2+ arguments): Loads current configs, modifies the key, and writes changes back to `~/.purrconfig`.

### 4.4 `purr add`

Walks directories concurrently and stages new or modified files in the `.purr` index.

```mermaid
sequenceDiagram
    actor Developer
    participant Core as internal/purrcommands/add.go
    participant WP as Worker Pool (Goroutines)
    participant Store as internal/hash
    participant OS as Filesystem

    Developer->>Core: Runs "purr add ."
    Core->>OS: ReadIndex(".purr/index")
    OS-->>Core: Return existing index entries
    Core->>OS: Walk workspace directories (ignoring hidden files)
    loop For each file discovered
        Core->>WP: Dispatch path to worker goroutine
        activate WP
        WP->>OS: Stat file (Mtime, Mode, Size)
        alt Stat Cache Match (Mtime == Index Mtime)
            WP-->>Core: Increment skipped count & exit
        else File modified or new
            WP->>OS: Read file content
            WP->>WP: Prepend "blob {size}\x00" & compute SHA-1
            WP->>Store: WriteBlobWithSHA()
            activate Store
            Store->>Store: zlib compress blob
            Store->>OS: StoreObject under .purr/objects/xx/yyyy...
            Store-->>WP: Return SHA-1 hash
            deactivate Store
            WP->>WP: Create PopulateAllIndexField entry
            WP-->>Core: Acquire Mutex & write to shared indexMap
        end
        deactivate WP
    end
    Core->>Core: Remove deleted files (in index but missing from walk)
    Core->>Core: Sort indexMap entries lexicographically by path
    Core->>OS: WriteIndex(".purr/index")
    Core-->>Developer: Display success summary (Added/Skipped/Removed)
```

1. **Directory Checks**: Core calls `AddPurrFiles(args...)` from `internal/purrcommands/add.go`, validating that the directory has been initialized with a `.purr` storage root.
2. **Workspace Traversal**:
   - **Staging All**: Walks the current directory recursively skipping hidden folders and `.purr` contents. Files present in the old index but missing from the disk are removed from the staging area.
   - **Staging Specific Paths**: Collects the files listed in the arguments, gracefully unstaging files if they have been deleted.
3. **Concurrent Hashing (Worker Pool)**: For modified or new files, tasks are distributed to a concurrent worker pool:
   - Calculates the `SHA-1` checksum of the file's raw content.
   - Writes a zlib-compressed blob object to `.purr/objects/XX/YYYY...` only if the file content has changed.
4. **Index Serialization**: Integrates new file entries, sorts the index collection alphabetically by path, and performs an atomic write to `.purr/index`.

### 4.5 `purr commit`

Generates an immutable commit snapshot containing the staged workspace states.

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/commit.go
    participant Core as internal/purrcommands/commit.go
    participant Util as internal/objects
    participant OS as Filesystem (.purr/)

    User->>CLI: Runs "purr commit -m <msg>"
    CLI->>Util: CheckConfigFile() (Validates user.name and user.email)
    Util-->>CLI: Return name and email
    CLI->>Core: CommitPurrFiles(message)

    activate Core
    Core->>OS: Check if ".purr" is initialized
    Core->>OS: Read index staged entries
    Core->>Util: BuildTreeObject() (Groups files & recursively serializes subtrees)
    activate Util
    Util->>OS: Compress & store nested child Tree objects
    Util-->>Core: Return root Tree SHA-1 hash
    deactivate Util
    Core->>OS: GetHEADCommit() (Resolves parent commit hash)
    alt Parent tree hash == New tree hash
        Core-->>User: Abort: "nothing to commit, working tree clean"
    else Changes detected
        Core->>Util: BuildCommitObject() (Formats plain-text commit header & body)
        Core->>OS: Compress & Store Commit object under objects/xx/yyyy...
        Core->>OS: UpdateHEAD() (Update active branch pointer)
        Core-->>User: Print short 7-char commit hash and message
    end
    deactivate Core
```

1. **Metadata Setup**: Extracts current stage data from `.purr/index` and fetches the parent commit reference by reading the local branch ref pointed to by `.purr/HEAD`.
2. **Tree Object Assembly**:
   - Recursively groups index files by their parent directories.
   - Assembles sub-tree objects containing nested entries.
   - Hashes and serializes all sub-tree directories.
   - Assembles the root Tree object linking files and sub-trees.
   - Computes the root Tree `SHA-1` hash.
3. **Deduplication Validation**: Compares the new Tree hash with the parent commit's Tree hash. If they are identical, the commit is aborted since no changes have been staged.
4. **Write Objects**:
   - Writes the compressed Tree object into the database.
   - Generates Commit metadata (Tree hash, Parent hash, Author name/email, message, and timestamp).
   - Computes the Commit `SHA-1` hash.
   - Writes the compressed Commit object into the database.
5. **Updating Refs**: Updates the target branch pointer (e.g., `.purr/refs/heads/main`) to point to the new commit's `SHA-1` hash.
