//go:build !windows

package platform

// SetHidden is a no-op on Unix systems. Files starting with '.' are
// already treated as hidden by convention.
func SetHidden(path string) error {
	return nil
}
