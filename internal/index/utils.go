package index

import (
	"persephone/internal/platform"
	"os"
)

func PopulateAllIndexField(fileInfo os.FileInfo, relPath string) IndexEntry {
	stat := platform.ExtractStat(fileInfo)
	return IndexEntry{
		Ctime: stat.Ctime,
		Mtime: fileInfo.ModTime(),
		Dev:   stat.Dev,
		Ino:   stat.Ino,
		Mode:  uint32(fileInfo.Mode()),
		Uid:   stat.Uid,
		Gid:   stat.Gid,
		Size:  uint32(fileInfo.Size()),
		Stage: 0,
		Path:  relPath,
	}
}
