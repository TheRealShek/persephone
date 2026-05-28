# Purr Commands: Implementation Guide & Architectural Flows

This document details the software design, sequence flows, and internal implementations of each custom **Purr** command.

---

## 1. `purr init`

Initializes a local repository with the necessary directory hierarchy and metadata configuration.

### Sequence Flow

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/init.go (Cobra)
    participant Core as internal/purrCommands/Init.go
    participant OS as Filesystem

    User->>CLI: Runs "purr init"
    CLI->>Core: InitPurrDirectories(".")
    
    activate Core
    Core->>OS: os.MkdirAll(".purr/{objects,refs/heads,logs}")
    Note over Core,OS: If OS is Windows, sets .purr directory as hidden
    
    Core->>OS: Write valid 12-byte header to ".purr/index"
    Note over Core,OS: Header: "DIRC" (4B) | Version 2 (4B) | Count 0 (4B)
    
    Core->>OS: Write "ref: refs/heads/main\n" to ".purr/HEAD"
    
    Core-->>CLI: Returns success status (nil)
    deactivate Core
    
    CLI-->>User: Prints "Initialized empty repository"
```

### Detailed Steps

1. **Invocation**: The user executes `purr init`. The runtime invokes the entrypoint in `cmd/init.go`.
2. **Directory Bootstrapping**: Core calls `InitPurrDirectories(".")` inside `internal/purrCommands/Init.go`. It builds:
   - `.purr/objects` (object store)
   - `.purr/refs/heads` (local refs)
   - `.purr/logs` (lifecycle history logs)
3. **OS-Specific Adjustments**: On Windows platforms, `.purr` is set to "hidden" using syscalls.
4. **Staging Index Creation**: Writes a valid 12-byte binary index header if the file is missing:
   - Magic signature: `"DIRC"` (4 bytes)
   - Staging Version: `2` (4 bytes, big-endian)
   - Initial count of entries: `0` (4 bytes, big-endian)
5. **HEAD Initialization**: Writes `"ref: refs/heads/main\n"` to `.purr/HEAD`, binding active tracking to the `main` branch.

---

## 2. `purr ls-files`

Lists all files currently tracked in the staging index.

### Sequence Flow

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/ls-files.go (Cobra)
    participant Core as internal/purrCommands/LsFiles.go
    participant Utils as internal/utils (Index Reader)

    User->>CLI: Runs "purr ls-files [--debug]"
    CLI->>Core: ListFiles(showDebug)
    
    activate Core
    Core->>Utils: ReadIndex(".purr/index")
    Utils-->>Core: Slice of Index entries
    
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

### Detailed Steps

1. **Invocation**: The user executes `purr ls-files [--debug]`.
2. **Loading Index**: The CLI calls `ListFiles(showDebug)` in `internal/purrCommands/LsFiles.go`. It reads the binary database under `.purr/index` using the `utils.ReadIndex` library helper.
3. **Empty Bounds Handling**: If the index contains `0` records, the command exits with `"No files in index"`.
4. **Output Rendering**:
   - **Default Mode**: Displays the calculated object hash, file mode, and relative path.
   - **Debug Mode**: Prints detailed binary index records, including timestamps (`mtime`, `ctime`), host attributes (`dev`, `ino`, `uid`, `gid`), file sizes, and stage parameters.

---

## 3. `purr config`

Manages configuration files on the local machine.

### Sequence Flow

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/config.go (Cobra)
    participant Core as internal/purrCommands/Config.go
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

### Detailed Steps

1. **Invocation**: The user executes `purr config <key> [value]`.
2. **CLI Routing**: Handles read or write modes depending on the argument length:
   - **Read Mode** (1 argument): Invokes `utils.ReadConfig()` to load the global configuration file (`~/.purrconfig`) and outputs the value of the requested key (typically `user.name` or `user.email`).
   - **Write Mode** (2+ arguments): Loads current configs (or builds a new configuration if missing), modifies the key, and writes the changes back to `~/.purrconfig`.

---

## 4. `purr add`

Walks directories concurrently and stages new or modified files in the `.purr` index.

### Sequence Flow

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/add.go (Cobra)
    participant Core as internal/purrCommands/Add.go
    participant WP as Worker Pool (Goroutines)
    participant OS as Filesystem

    User->>CLI: Runs "purr add ." or "purr add <file>"
    CLI->>Core: AddPurrFiles(args...)
    
    activate Core
    Core->>OS: Check if ".purr" folder exists
    
    alt Add All ("add .")
        Core->>OS: Walk directory tree recursively
        OS-->>Core: Return file paths (skipping hidden objects)
    else Add Specific Files
        Core->>Core: Filter out invalid / out-of-bounds files
    end
    
    loop For each eligible file (Concurrent Worker Pool)
        Core->>WP: Spawn hash and compress task
        WP->>OS: Compute file hash & write compressed zlib blob
        WP-->>Core: Return generated blob SHA-1 and file size
    end
    
    Core->>Core: Sort updated index entries alphabetically
    Core->>OS: Write new serialized index to ".purr/index"
    
    Core-->>CLI: Return success status
    deactivate Core
    CLI-->>User: Displays staging results summary
```

### Detailed Steps

1. **Invocation**: The user runs `purr add .` or `purr add file1.txt`.
2. **Directory Checks**: Core calls `AddPurrFiles(args...)` from `internal/purrCommands/Add.go`, validating that the directory has been initialized with a `.purr` storage root.
3. **Workspace Traversal**:
   - **Staging All**: Walks the current directory recursively using optimized walk steps that skip hidden folders and `.purr` contents.
   - **Staging Specific Paths**: Collects the files listed in the arguments, filtering out missing objects, folders, and out-of-bounds files.
4. **Concurrent Hashing (Worker Pool)**: For modified or new files, tasks are distributed to a concurrent worker pool:
   - Calculates the `SHA-1` checksum of the file's raw content.
   - Writes a zlib-compressed blob object to `.purr/objects/XX/YYYY...` only if the file content has changed.
5. **Index Serialization**: Integrates new file entries, sorts the index collection alphabetically by path, and performs an atomic write to `.purr/index`.

---

## 5. `purr commit`

Generates an immutable commit snapshot containing the staged workspace states.

### Sequence Flow

```mermaid
sequenceDiagram
    actor User
    participant CLI as cmd/commit.go (Cobra)
    participant Core as internal/purrCommands/Commit.go
    participant OS as Filesystem (.purr/)

    User->>CLI: Runs "purr commit -m <msg>"
    CLI->>Core: CommitPurrFiles(message)
    
    activate Core
    Core->>OS: Check if ".purr" is initialized
    Core->>OS: Read current index entries & active HEAD pointer
    
    Core->>Core: Convert index records into Tree entries
    Core->>Core: Serialize and compute Tree SHA-1
    
    opt Parent commit exists
        Core->>OS: Read parent commit's Tree hash
        alt Tree hashes match (No modifications)
            Core-->>CLI: Print "No changes to commit" (Aborts execution)
        end
    end
    
    Core->>OS: Write zlib-compressed Tree object to "objects/"
    
    Core->>Core: Build Commit metadata (Tree hash, parent, author, message, timestamp)
    Core->>Core: Compute Commit SHA-1
    Core->>OS: Write zlib-compressed Commit object to "objects/"
    
    Core->>OS: Update refs/heads/<branch> or HEAD with new Commit SHA-1
    
    Core-->>CLI: Returns new Commit SHA-1
    deactivate Core
    CLI-->>User: Displays successful Commit SHA-1
```

### Detailed Steps

1. **Invocation**: The user executes `purr commit -m "commit message"`.
2. **Metadata Setup**: Extracts current stage data from `.purr/index` and fetches the parent commit reference by reading the local branch ref pointed to by `.purr/HEAD`.
3. **Tree Object Assembly**:
   - Groups index files into directory entries.
   - Serializes folders into standard Tree format entries.
   - Computes the Tree `SHA-1` hash.
4. **Deduplication Validation**: Compares the new Tree hash with the parent commit's Tree hash. If they are identical, the commit is aborted since no changes have been staged.
5. **Write Objects**:
   - Writes the compressed Tree object into the database.
   - Generates Commit metadata (Tree hash, Parent hash, Author name/email, message, and timestamp).
   - Computes the Commit `SHA-1` hash.
   - Writes the compressed Commit object into the database.
6. **Updating Refs**: Updates the target branch pointer (e.g., `.purr/refs/heads/main`) to point to the new commit's `SHA-1` hash.