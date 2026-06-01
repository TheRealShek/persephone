package utils

import "time"

// IndexEntry represents a single file entry staged in the VCS index.
// It acts as a cache of filesystem stat metadata to enable fast modification checks
// without reading and re-hashing file content. By comparing this stat data against the
// actual file metadata, the VCS can quickly identify new, modified, or deleted files.
// The fields align closely with Git's internal index layout to maintain conceptual parity.
type IndexEntry struct {
	// Stat cache metadata for change detection
	Ctime time.Time // File status change time (metadata modification)
	Mtime time.Time // File content modification time
	Dev   uint32    // Device ID where the file resides
	Ino   uint32    // File inode number for unique file identification on the filesystem
	Mode  uint32    // File permissions and type (e.g., regular file, executable)
	Uid   uint32    // User ID of the file owner
	Gid   uint32    // Group ID of the file owner
	Size  uint32    // File size in bytes (crucial for rapid modification detection)

	// Content addressable identifier
	Sha1 [20]byte // 160-bit SHA-1 hash of the zlib-compressed file content

	// VCS flags
	Stage uint16 // 0 represents normal staged state; 1, 2, and 3 are reserved for merge conflict stages

	// Repository context
	Path string // File path relative to the repository root directory (e.g. "pkg/foo.go")
}

// PurrConfig defines the global identity configuration (author/committer credentials).
// It is read and written in JSON format under the user's home directory (~/.purrconfig)
// to bypass complex INI parsing or per-repository config overhead, simplifying multi-repo setups.
type PurrConfig struct {
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// TreeEntries represents a single directory or file entry structured for Git-style tree object building.
// These entries are sorted lexicographically (directories appended with a trailing '/' during comparison)
// to match Git's tree formatting constraints exactly, ensuring deterministic hash generation.
type TreeEntries struct {
	Name     string // Base name of the file or directory
	Filename string // Full relative path of the file
	Sha1Hex  string // Hexadecimal representation of the object's SHA-1 hash
	IsTree   bool   // True if the entry represents a subdirectory (tree), false for a file (blob)
	Mode     string // Git-compatible file/tree mode string (e.g., "100644", "100755", "040000")
}

// Index serves as a in-memory representation of the staged files.
type Index struct {
	Entries []IndexEntry
}

// CommitObj is the in-memory representation of commit metadata.
// BuildCommitObject serializes it into a Git-style plain-text payload before zlib compression.
type CommitObj struct {
	TreeHash   string     // SHA-1 hash of the root tree object representing the repository state
	ParentHash string     // SHA-1 hash of the parent commit (empty string for initial commits)
	Author     PurrConfig // Creator of the changes
	Committer  PurrConfig // Person who committed the changes (usually identical to the author)
	Message    string     // Developer-provided commit message
	Timestamp  time.Time  // Time of commit generation, preserved for deterministic hashing
}
