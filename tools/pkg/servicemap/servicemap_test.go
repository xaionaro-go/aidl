package servicemap

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildServiceMap(t *testing.T) {
	frameworksBase := "../3rdparty/frameworks-base"
	m, err := BuildServiceMap(frameworksBase)
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("3rdparty submodules not available")
	}
	require.NoError(t, err)
	require.Greater(t, len(m), 50)

	entry, ok := m["power"]
	require.True(t, ok, "power service not found")
	require.Equal(t, "android.os.IPowerManager", entry.AIDLDescriptor)
	require.Equal(t, "POWER_SERVICE", entry.ConstantName)
	require.Equal(t, "IPowerManager", entry.AIDLInterface)
	require.Equal(t, "power", entry.ServiceName)

	entry, ok = m["account"]
	require.True(t, ok, "account service not found")
	require.Equal(t, "android.accounts.IAccountManager", entry.AIDLDescriptor)
	require.Equal(t, "ACCOUNT_SERVICE", entry.ConstantName)

	// ACTIVITY_SERVICE registers ActivityManager directly (no Stub.asInterface),
	// so it should NOT appear in the service map.
	_, ok = m["activity"]
	require.False(t, ok, "activity service should not be in map (no AIDL binder interface)")
}

func TestBuildServiceMapMissingDir(t *testing.T) {
	_, err := BuildServiceMap("/nonexistent/path")
	require.Error(t, err)
	require.True(t, errors.Is(err, os.ErrNotExist))
}

func TestExtractImports(t *testing.T) {
	src := `package android.app;

import android.os.IPowerManager;
import android.accounts.IAccountManager;
import android.content.Context;
import static android.app.Flags.enableFeature;
`

	imports := extractImports(src)
	require.Equal(t, "android.os.IPowerManager", imports["IPowerManager"])
	require.Equal(t, "android.accounts.IAccountManager", imports["IAccountManager"])
	require.Equal(t, "android.content.Context", imports["Context"])

	// Static imports are not matched by the regex because "static" is
	// captured as the first word token; this is fine since we only need
	// regular imports for AIDL interface resolution.
	_, ok := imports["enableFeature"]
	require.False(t, ok)
}
