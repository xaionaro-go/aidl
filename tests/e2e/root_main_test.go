//go:build e2e_root

package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Root tests REQUIRE UID 0 (adb root) and permissive SELinux
	// (setenforce 0) to bypass service_manager restrictions for HALs
	// and privileged system services.
	//
	// WARNING: These tests intentionally skip the root-refusal guard in
	// bindercli_test.go. Only read-only queries are performed — no
	// destructive operations (e.g. DeleteAllKeys, WipeData) are called.
	// Review every new root test for safety before adding it here.
	if os.Getuid() != 0 {
		os.Stderr.WriteString("FATAL: e2e_root tests require root (UID 0).\n")
		os.Stderr.WriteString("Run: adb root && adb shell setenforce 0\n")
		os.Stderr.WriteString("Then push and execute the e2e_root binary.\n")
		os.Exit(1)
	}

	if _, err := os.Stat("/dev/binder"); err != nil {
		os.Stderr.WriteString("FATAL: /dev/binder not found — must run on device.\n")
		os.Exit(1)
	}

	onDevice = true
	os.Exit(m.Run())
}
