package utils

import "time"

type IndexEntry struct {
	// Stat cache (for fast change detection)
	Ctime time.Time // Change time
	Mtime time.Time // Modification time
	Dev   uint32    // Device ID
	Ino   uint32    // Inode number
	Mode  uint32    // File permissions
	Uid   uint32    // User ID
	Gid   uint32    // Group ID
	Size  uint32    // File size in bytes
	// Content identification
	Sha1 [20]byte // SHA-1 hash of content
	// Flags
	Stage uint16 // 0=normal, 1-3=conflict stages
	// Path
	Path string // Relative path from repo root
}

// PurrConfig stores user configuration settings (Author)
type PurrConfig struct {
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// Clean Approach for func ComputeTreeSHA1()
type TreeEntries struct {
	Name     string
	Filename string
	Sha1Hex  string // hex string
	IsTree   bool   // for sorting (directory if true)
	Mode     string // file mode (e.g., "100644", "100755", "040000")
}

//
type Index struct {
	Entries []IndexEntry
}

// CommitObj represents a commit object in the repository
type CommitObj struct {
	TreeHash   string     `json:"tree"`      // Hash of the tree object
	ParentHash string     `json:"parent"`    // Hash of parent commit(s)
	Author     PurrConfig `json:"author"`    // Author information
	Committer  PurrConfig `json:"committer"` // Committer information (usually same as author)
	Message    string     `json:"message"`   // Commit message
}
