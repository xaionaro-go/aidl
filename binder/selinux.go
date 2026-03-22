//go:build linux

package binder

import (
	"errors"
	"fmt"
	"os"

	selinux "github.com/opencontainers/selinux/go-selinux"

	aidlerrors "github.com/AndroidGoLab/binder/errors"
)

// enrichWithSELinuxContext checks whether err is a permission-related binder
// error (AIDL SecurityException or kernel EPERM) and, if so, wraps it with
// the current process's SELinux context label. This makes permission denials
// immediately diagnosable without requiring a separate `adb shell ps -eZ`
// lookup.
func enrichWithSELinuxContext(err error) error {
	if err == nil {
		return nil
	}

	if !isPermissionError(err) {
		return err
	}

	if !selinux.GetEnabled() {
		return err
	}

	label, labelErr := selinux.CurrentLabel()
	if labelErr != nil {
		return err
	}

	if label == "" {
		return err
	}

	return fmt.Errorf("SELinux context %q: %w", label, err)
}

// isPermissionError returns true when err represents a binder permission
// denial — either an AIDL-level SecurityException or a kernel-level EPERM
// wrapped in a BinderError.
func isPermissionError(err error) bool {
	var statusErr *aidlerrors.StatusError
	if errors.As(err, &statusErr) {
		return statusErr.Exception == aidlerrors.ExceptionSecurity
	}

	var binderErr *aidlerrors.BinderError
	if errors.As(err, &binderErr) {
		return errors.Is(binderErr.Err, os.ErrPermission)
	}

	return false
}
