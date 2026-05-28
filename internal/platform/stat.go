package platform

import "time"

// StatData holds platform-extracted file metadata for index entries.
type StatData struct {
	Ctime time.Time
	Dev   uint32
	Ino   uint32
	Uid   uint32
	Gid   uint32
}
