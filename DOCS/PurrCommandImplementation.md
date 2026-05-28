

## `purr init` Command: Summary and Detailed Flow

### Summary Flow

1. **User runs:** `purr init`
2. **CLI calls:** `InitPurrDirectories(".")`
3. **InitPurrDirectories:**
		- Creates `.purr` directory and subdirectories
		- (Windows) Sets `.purr` as hidden
		- Creates `.purr/index` with valid header if missing
		- Creates `.purr/HEAD` pointing to `refs/heads/main` if missing
4. **CLI prints:** Success or error message

---

### Detailed Step-by-Step Flow

**1. User runs:** `purr init`

	 - CLI entrypoint: `cmd/init.go`
	 - Cobra command's `Run` function is executed

**2. Calls:** `purrCommands.InitPurrDirectories(".")`  
	 *(Defined in `internal/purrCommands/Init.go`)*

	 **What `InitPurrDirectories` does:**

	 - **Creates the `.purr` directory structure:**
		 - `.purr/objects`
		 - `.purr/refs/heads`
		 - `.purr/logs`
		 - Uses `os.MkdirAll` to ensure all directories exist

	 - **On Windows:**
		 - Sets `.purr` as hidden using Windows system calls (if not already hidden)

	 - **Creates the index file:**
		 - Checks if `.purr/index` exists
		 - If not, writes a valid 12-byte header:
			 - `"DIRC"` (magic, 4 bytes)
			 - Version `2` (4 bytes, big-endian)
			 - Entry count `0` (4 bytes, big-endian)
		 - Uses `os.WriteFile` to write the header

	 - **Creates the HEAD file:**
		 - Checks if `.purr/HEAD` exists
		 - If not, writes `ref: refs/heads/main\n` to it, pointing HEAD to the main branch

	 - **Returns:**
		 - If all steps succeed, returns `nil` (success)
		 - If any step fails, returns an error

**3. Back in the CLI (`cmd/init.go`):**

	 - If `InitPurrDirectories` returns no error, prints: `Initialized empty repository`
	 - If there is an error, prints the error message

---

**This is the complete function call and logic flow for `purr init`.**


## `purr ls-files` Command: Summary and Detailed Flow

### Summary Flow

1. **User runs:** `purr ls-files` (optionally with `--debug`)
2. **CLI calls:** `purrCommands.ListFiles(showDebug)`
3. **ListFiles:**
		- Reads `.purr/index` using `utils.ReadIndex`
		- If index is empty, prints a message and exits
		- If not, prints a list of staged files:
				- Simple mode: SHA-1, mode, and path for each file
				- Debug mode: detailed metadata for each file
4. **CLI prints:** Output to the user

---

### Detailed Step-by-Step Flow

**1. User runs:** `purr ls-files [--debug]`

	 - CLI entrypoint: `cmd/ls-files.go`
	 - Cobra command's `Run` function is executed
	 - `--debug` flag is parsed (default: false)

**2. Calls:** `purrCommands.ListFiles(showDebug)`  
	 *(Defined in `internal/purrCommands/LsFiles.go`)*

	 **What `ListFiles` does:**

	 - **Reads the index:**
		 - Constructs the path `.purr/index`
		 - Calls `utils.ReadIndex(indexPath)` to get a slice of index entries
		 - If reading fails, returns an error

	 - **Handles empty index:**
		 - If there are no entries, prints "No files in index" and returns

	 - **Prints file information:**
		 - If `showDebug` is true:
				 - Prints detailed metadata for each file (path, SHA1, mode, size, mtime, ctime, dev, ino, uid, gid, stage)
		 - If `showDebug` is false:
				 - Prints a simple list: SHA1, mode, and path for each file

	 - **Returns:**
		 - Returns `nil` on success, or an error if something failed

**3. Back in the CLI (`cmd/ls-files.go`):**

	 - If `ListFiles` returns no error, nothing further is printed
	 - If there is an error, prints the error message

---

**This is the complete function call and logic flow for `purr ls-files`.**


## `purr config` Command: Summary and Detailed Flow

### Summary Flow

1. **User runs:** `purr config <key> [value]`
2. **CLI calls:** `purrCommands.ConfigCommand(args...)`
3. **ConfigCommand:**
		- If only `<key>` is provided, reads the config value and prints it
		- If both `<key>` and `<value>` are provided, updates the config and saves it
		- Uses helper functions to read/write config from the user's home directory
4. **CLI prints:** Result or error message

---

### Detailed Step-by-Step Flow

**1. User runs:** `purr config <key> [value]`

	 - CLI entrypoint: `cmd/config.go`
	 - Cobra command's `Run` function is executed
	 - Arguments are passed as `args` to the handler

**2. Calls:** `purrCommands.ConfigCommand(args...)`  
	 *(Defined in `internal/purrCommands/Config.go`)*

	 **What `ConfigCommand` does:**

	 - **Argument parsing:**
		 - If no arguments, prints usage and returns an error
		 - If one argument, enters read mode
		 - If two or more arguments, enters write mode (joins all after the key as the value)

	 - **Read mode:**
		 - Calls `utils.ReadConfig()` to load the config from the user's home directory (`~/.purrconfig`)
		 - Prints the value for the requested key (`user.name` or `user.email`)
		 - If the key is unknown, prints an error

	 - **Write mode:**
		 - Calls `utils.ReadConfig()` to load the config (or creates a new one if missing)
		 - Updates the value for the requested key
		 - Calls `utils.WriteConfig()` to save the updated config back to `~/.purrconfig`
		 - Prints confirmation of the change
		 - If the key is unknown, prints an error

	 - **Returns:**
		 - Returns `nil` on success, or an error if something failed

**3. Back in the CLI (`cmd/config.go`):**

	 - If `ConfigCommand` returns no error, nothing further is printed
	 - If there is an error, prints the error message

---

**This is the complete function call and logic flow for `purr config`.**



## `purr add` Command: Summary and Detailed Flow

### Summary Flow

1. **User runs:** `purr add .` or `purr add <file1> <file2> ...`
2. **CLI calls:** `purrCommands.AddPurrFiles(args...)`
3. **AddPurrFiles:**
		- Checks if `.purr` directory exists (repo initialized)
		- If `purr add .`, stages all non-hidden files recursively (concurrent, skips unchanged)
		- If `purr add <files>`, stages only specified files (concurrent, skips unchanged/hidden)
		- For each new/modified file: creates a blob object, updates the index
		- Writes the updated index to disk
4. **CLI prints:** Summary of added/skipped files or errors

---

### Detailed Step-by-Step Flow

**1. User runs:** `purr add .` or `purr add <file1> <file2> ...`

	 - CLI entrypoint: `cmd/add.go`
	 - Cobra command's `Run` function is executed
	 - Arguments are passed as `args` to the handler

**2. Calls:** `purrCommands.AddPurrFiles(args...)`  
	 *(Defined in `internal/purrCommands/Add.go`)*

	 **What `AddPurrFiles` does:**

	 - **Checks repository initialization:**
		 - Verifies `.purr` directory exists using `utils.ExistsAndIsDirectory`
		 - If not, prints error and exits

	 - **Handles arguments:**
		 - If no files specified, prints "No Files added" and returns
		 - If argument is `.`, calls `addAllPurrFiles` to stage all files
		 - Otherwise, calls `addSpecificPurrFiles` to stage only listed files

	 - **addAllPurrFiles:**
		 - Loads current index entries from `.purr/index`
		 - Recursively walks the working directory, skipping hidden files/dirs
		 - For each file:
				 - If new or modified, creates a blob (calls `utils.WriteBlobWithSHA`), updates index entry
				 - Uses goroutines and a worker pool for concurrency
		 - After all files processed, writes updated index (sorted) to disk

	 - **addSpecificPurrFiles:**
		 - Loads current index entries from `.purr/index`
		 - For each specified file:
				 - Skips hidden files/dirs, directories, or files outside repo
				 - If new or modified, creates a blob, updates index entry
				 - Uses goroutines and a worker pool for concurrency
		 - After all files processed, writes updated index (sorted) to disk if any files were added

	 - **Returns:**
		 - Returns `nil` on success, or an error if something failed

**3. Back in the CLI (`cmd/add.go`):**

	 - If `AddPurrFiles` returns no error, prints "Files added to index"
	 - If there is an error, prints the error message

---

**This is the complete function call and logic flow for `purr add`.**


## `purr commit` Command: Summary and Detailed Flow

### Summary Flow

1. **User runs:** `purr commit -m "<message>"`
2. **CLI calls:** `purrCommands.CommitPurrFiles(message)`
3. **CommitPurrFiles:**
	 - Checks if `.purr` directory exists (repo initialized)
	 - Reads the index and HEAD commit
	 - Builds a tree object from the index
	 - Checks for duplicate commit (compares new tree hash with parent commit's tree hash)
	 - If not duplicate:
		 - Creates and writes the tree object
		 - Builds and writes the commit object (with parent, author, message)
		 - Updates HEAD to new commit
	 - If duplicate:
		 - Prints message and aborts commit
4. **CLI prints:** Commit SHA or error/duplicate message

---

### Detailed Step-by-Step Flow

**1. User runs:** `purr commit -m "<message>"`

	 - CLI entrypoint: `cmd/commit.go`
	 - Cobra command's `Run` function is executed
	 - Commit message is parsed from `-m` flag

**2. Calls:** `purrCommands.CommitPurrFiles(message)`  
	 *(Defined in `internal/purrCommands/Commit.go`)*

	 **What `CommitPurrFiles` does:**

	 - **Checks repository initialization:**
		 - Verifies `.purr` directory exists using `utils.ExistsAndIsDirectory`
		 - If not, prints error and exits

	 - **Reads index and HEAD:**
		 - Loads index entries from `.purr/index` (calls `utils.ReadIndex`)
		 - Reads current HEAD reference from `.purr/HEAD` (calls `utils.ReadHEAD`)
		 - If HEAD points to a branch, reads the latest commit SHA from the branch ref
		 - If HEAD is detached, uses the SHA directly

	 - **Builds tree object:**
		 - Converts index entries to tree entries (calls `getTreeEntries`)
		 - Serializes tree object (calls `BuildTreeObject`)
		 - Computes tree SHA-1 (calls `ComputeTreeSHA1`)

	 - **Checks for duplicate commit:**
		 - If there is a parent commit:
			 - Reads parent commit object (calls `GetCommitTreeHash`)
			 - Compares new tree hash with parent commit's tree hash
			 - If hashes match, prints "No changes to commit" and aborts

	 - **Creates and writes tree object:**
		 - Compresses tree object (zlib)
		 - Writes to `.purr/objects/<treeSHA>`

	 - **Builds and writes commit object:**
		 - Constructs commit object (calls `BuildCommitObject`)
		 - Computes commit SHA-1 (calls `ComputeCommitSHA1`)
		 - Compresses and writes commit object to `.purr/objects/<commitSHA>`

	 - **Updates HEAD:**
		 - Updates branch ref in `.purr/refs/heads/<branch>` to new commit SHA
		 - If HEAD is detached, updates `.purr/HEAD` directly

	 - **Returns:**
		 - On success, returns new commit SHA
		 - On error, returns error message

**3. Back in the CLI (`cmd/commit.go`):**

	 - If `CommitPurrFiles` returns a commit SHA, prints: `Committed as <commitSHA>`
	 - If duplicate, prints: `No changes to commit`
	 - If there is an error, prints the error message

---

**This is the complete function call and logic flow for `purr commit`.**