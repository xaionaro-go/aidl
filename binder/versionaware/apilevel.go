package versionaware

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// detectAPILevel returns the Android API level of the running device.
// It tries `getprop ro.build.version.sdk` first (works for any user),
// then falls back to parsing /system/build.prop (requires root).
// Returns 0 if detection fails (e.g. when running outside Android).
func detectAPILevel() int {
	if n := detectViaGetprop(); n > 0 {
		return n
	}
	return detectViaBuildProp()
}

// getpropPaths lists candidate locations for the getprop binary.
// The full path is needed because the binary's PATH may not include
// /system/bin when launched from /data/local/tmp.
var getpropPaths = []string{
	"/system/bin/getprop",
	"getprop",
}

func detectViaGetprop() int {
	for _, path := range getpropPaths {
		out, err := exec.Command(path, "ro.build.version.sdk").Output()
		if err != nil {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			continue
		}
		return n
	}
	return 0
}

func detectViaBuildProp() int {
	data, err := os.ReadFile("/system/build.prop")
	if err != nil {
		return 0
	}

	const prefix = "ro.build.version.sdk="
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		val := strings.TrimPrefix(line, prefix)
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return 0
		}
		return n
	}

	return 0
}
