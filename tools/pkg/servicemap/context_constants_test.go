package servicemap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractContextConstants(t *testing.T) {
	src := `class C {
    public static final String POWER_SERVICE = "power";
    public static final String ACTIVITY_SERVICE = "activity";
    public static final int BIND_FOREGROUND_SERVICE = 0x04000000;
    public static final String WINDOW_SERVICE = "window";
}`

	constants := ExtractContextConstants(src)
	require.Len(t, constants, 3)
	require.Equal(t, "power", constants["POWER_SERVICE"])
	require.Equal(t, "activity", constants["ACTIVITY_SERVICE"])
	require.Equal(t, "window", constants["WINDOW_SERVICE"])
}

func TestExtractContextConstantsReal(t *testing.T) {
	src, err := os.ReadFile("../3rdparty/frameworks-base/core/java/android/content/Context.java")
	if err != nil {
		t.Skip("3rdparty submodules not available")
	}

	constants := ExtractContextConstants(string(src))

	// Should find many service constants
	require.Greater(t, len(constants), 100)

	// Verify known ones
	require.Equal(t, "power", constants["POWER_SERVICE"])
	require.Equal(t, "activity", constants["ACTIVITY_SERVICE"])
}
