//go:build windows

package platform

import (
	"os"
	"syscall"
	"time"
)

// ExtractStat extracts platform-specific stat data from os.FileInfo.
func ExtractStat(fileInfo os.FileInfo) StatData {
	stat := fileInfo.Sys().(*syscall.Win32FileAttributeData)
	return StatData{
		Ctime: time.Unix(0, stat.CreationTime.Nanoseconds()),
		Dev:   0, // Not applicable on Windows
		Ino:   0, // Not applicable on Windows
		Uid:   0, // Not applicable on Windows
		Gid:   0, // Not applicable on Windows
	}
}
