package index

import "time"

type IndexEntry struct {
	Ctime time.Time
	Mtime time.Time
	Dev   uint32
	Ino   uint32
	Mode  uint32
	Uid   uint32
	Gid   uint32
	Size  uint32
	Sha1  [20]byte
	Stage uint16
	Path  string
}

type Index struct {
	Entries []IndexEntry
}
