//go:build linux

package platform

import (
	"os"
	"syscall"
	"time"
)

// ExtractStat extracts platform-specific stat data from os.FileInfo.
func ExtractStat(fileInfo os.FileInfo) StatData {
	stat := fileInfo.Sys().(*syscall.Stat_t)
	return StatData{
		Ctime: time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec),
		Dev:   uint32(stat.Dev),
		Ino:   uint32(stat.Ino),
		Uid:   stat.Uid,
		Gid:   stat.Gid,
	}
}
