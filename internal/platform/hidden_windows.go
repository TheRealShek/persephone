//go:build windows

package platform

import "syscall"

// SetHidden sets the hidden file attribute on Windows.
func SetHidden(path string) error {
	purrDirPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}

	attrs, err := syscall.GetFileAttributes(purrDirPtr)
	if err != nil {
		return err
	}

	// Only set if not already hidden
	if attrs&syscall.FILE_ATTRIBUTE_HIDDEN == 0 {
		err = syscall.SetFileAttributes(purrDirPtr, attrs|syscall.FILE_ATTRIBUTE_HIDDEN)
		if err != nil {
			return err
		}
	}

	return nil
}
