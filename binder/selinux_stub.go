//go:build !linux

package binder

// enrichWithSELinuxContext is a no-op on non-Linux platforms where SELinux
// is not available.
func enrichWithSELinuxContext(err error) error {
	return err
}
