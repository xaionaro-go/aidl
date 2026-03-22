//go:build linux

package binder

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	aidlerrors "github.com/AndroidGoLab/binder/errors"
)

func TestEnrichWithSELinuxContext_SecurityException(t *testing.T) {
	original := &aidlerrors.StatusError{
		Exception: aidlerrors.ExceptionSecurity,
		Message:   "caller lacks permission",
	}

	enriched := enrichWithSELinuxContext(original)
	require.Error(t, enriched)

	// On a host without SELinux (typical CI), the error passes through
	// unchanged because GetEnabled() returns false. On an SELinux-enabled
	// host the error message will contain "SELinux context".
	// Either way the original error must be reachable via Unwrap.
	var statusErr *aidlerrors.StatusError
	require.True(t, errors.As(enriched, &statusErr),
		"enriched error must unwrap to *StatusError")
	assert.Equal(t, aidlerrors.ExceptionSecurity, statusErr.Exception)
	assert.Equal(t, "caller lacks permission", statusErr.Message)
}

func TestEnrichWithSELinuxContext_BinderEPERM(t *testing.T) {
	original := &aidlerrors.BinderError{
		Op:  "ioctl(BINDER_WRITE_READ)",
		Err: syscall.EPERM,
	}

	enriched := enrichWithSELinuxContext(original)
	require.Error(t, enriched)

	var binderErr *aidlerrors.BinderError
	require.True(t, errors.As(enriched, &binderErr),
		"enriched error must unwrap to *BinderError")
	assert.Equal(t, "ioctl(BINDER_WRITE_READ)", binderErr.Op)
	assert.True(t, errors.Is(enriched, os.ErrPermission))
}

func TestEnrichWithSELinuxContext_NonPermissionPassthrough(t *testing.T) {
	original := &aidlerrors.StatusError{
		Exception: aidlerrors.ExceptionIllegalArgument,
		Message:   "bad arg",
	}

	enriched := enrichWithSELinuxContext(original)
	// Non-permission errors must pass through unchanged.
	assert.Equal(t, original, enriched,
		"non-permission error must not be wrapped")
}

func TestEnrichWithSELinuxContext_GenericErrorPassthrough(t *testing.T) {
	original := fmt.Errorf("something unrelated")

	enriched := enrichWithSELinuxContext(original)
	assert.Equal(t, original, enriched,
		"generic error must not be wrapped")
}

func TestEnrichWithSELinuxContext_NilPassthrough(t *testing.T) {
	assert.NoError(t, enrichWithSELinuxContext(nil))
}

func TestEnrichWithSELinuxContext_BinderNonEPERMPassthrough(t *testing.T) {
	original := &aidlerrors.BinderError{
		Op:  "mmap",
		Err: syscall.ENOMEM,
	}

	enriched := enrichWithSELinuxContext(original)
	assert.Equal(t, original, enriched,
		"BinderError with non-EPERM errno must not be wrapped")
}
