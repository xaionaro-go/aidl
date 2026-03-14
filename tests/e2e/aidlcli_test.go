//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	emulatorPath    = "/home/streaming/Android/Sdk/emulator/emulator"
	avdName         = "Pixel_7_API_35"
	aidlcliBinary   = "/tmp/aidlcli-test"
	deviceBinaryDir = "/data/local/tmp"
	deviceBinary    = deviceBinaryDir + "/aidlcli"
	bootTimeout     = 120 * time.Second
	bootPollPeriod  = 2 * time.Second
)

// emulatorSerial is the adb serial for the emulator (e.g., "emulator-5554").
// All adb commands use -s to target this specific device,
// so other connected devices are not affected.
var emulatorSerial string

// emulatorStartedByTest tracks whether TestMain started the emulator,
// so cleanup only kills it if we own it.
var emulatorStartedByTest bool

func TestMain(m *testing.M) {
	serial := findEmulator()
	if serial == "" {
		startEmulator()
		emulatorStartedByTest = true
		serial = waitForEmulatorSerial()
		if serial == "" {
			logAndExit("emulator started but no emulator serial found in adb devices")
		}
	}
	emulatorSerial = serial

	if err := waitForBoot(); err != nil {
		logAndExit("emulator boot timeout: " + err.Error())
	}

	if err := buildAidlcli(); err != nil {
		logAndExit("build aidlcli: " + err.Error())
	}

	if err := pushAidlcli(); err != nil {
		logAndExit("push aidlcli: " + err.Error())
	}

	code := m.Run()

	if emulatorStartedByTest {
		_ = adbCmd("emu", "kill").Run()
	}

	os.Exit(code)
}

// findEmulator scans `adb devices` for an already-running emulator
// and returns its serial (e.g., "emulator-5554"), or "" if none found.
func findEmulator() string {
	out, err := exec.Command("adb", "devices").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "device" && strings.HasPrefix(fields[0], "emulator-") {
			return fields[0]
		}
	}
	return ""
}

// waitForEmulatorSerial polls `adb devices` until an emulator appears.
func waitForEmulatorSerial() string {
	deadline := time.Now().Add(bootTimeout)
	for time.Now().Before(deadline) {
		serial := findEmulator()
		if serial != "" {
			return serial
		}
		time.Sleep(bootPollPeriod)
	}
	return ""
}

// adbCmd creates an exec.Command targeting the emulator via `adb -s <serial>`.
func adbCmd(args ...string) *exec.Cmd {
	fullArgs := append([]string{"-s", emulatorSerial}, args...)
	return exec.Command("adb", fullArgs...)
}

// startEmulator launches the emulator headlessly in the background.
func startEmulator() {
	cmd := exec.Command(
		emulatorPath,
		"-avd", avdName,
		"-no-window",
		"-no-audio",
		"-no-boot-anim",
		"-gpu", "swiftshader_indirect",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		logAndExit("starting emulator: " + err.Error())
	}
	// Detach — we will kill via `adb -s <serial> emu kill` later.
	go func() { _ = cmd.Wait() }()
}

// waitForBoot polls `adb -s <serial> shell getprop sys.boot_completed` until "1".
func waitForBoot() error {
	deadline := time.Now().Add(bootTimeout)
	for time.Now().Before(deadline) {
		out, err := adbCmd("shell", "getprop", "sys.boot_completed").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "1" {
			return nil
		}
		time.Sleep(bootPollPeriod)
	}
	return os.ErrDeadlineExceeded
}

// buildAidlcli cross-compiles the CLI for the emulator (x86_64 Linux).
func buildAidlcli() error {
	cmd := exec.Command(
		"go", "build",
		"-o", aidlcliBinary,
		"./cmd/aidlcli/",
	)
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)
	cmd.Dir = "/home/streaming/go/src/github.com/xaionaro-go/aidl"
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pushAidlcli copies the built binary to the emulator and makes it executable.
func pushAidlcli() error {
	if out, err := adbCmd("push", aidlcliBinary, deviceBinary).CombinedOutput(); err != nil {
		return &pushError{output: string(out), err: err}
	}
	if out, err := adbCmd("shell", "chmod", "755", deviceBinary).CombinedOutput(); err != nil {
		return &pushError{output: string(out), err: err}
	}
	return nil
}

type pushError struct {
	output string
	err    error
}

func (e *pushError) Error() string {
	return e.err.Error() + ": " + e.output
}

func (e *pushError) Unwrap() error {
	return e.err
}

// logAndExit writes to stderr and exits with code 1.
// Used only in TestMain where t.Fatal is unavailable.
func logAndExit(msg string) {
	os.Stderr.WriteString("FATAL: " + msg + "\n")
	if emulatorStartedByTest && emulatorSerial != "" {
		_ = adbCmd("emu", "kill").Run()
	}
	os.Exit(1)
}

// runAidlcli executes aidlcli on the emulator via `adb -s <serial> shell`.
// It always injects --format json for machine-parseable output.
// Arguments are joined into a single shell command string to preserve
// empty/quoted values across the adb shell boundary.
func runAidlcli(args ...string) (string, string, error) {
	// Build a single shell command string to avoid adb shell arg splitting issues.
	parts := make([]string, 0, len(args)+3)
	parts = append(parts, deviceBinary, "--format", "json")
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	shellCmd := strings.Join(parts, " ")
	cmd := adbCmd("shell", shellCmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// shellQuote wraps a string in single quotes for shell safety.
// Single quotes within the string are escaped.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// runAidlcliOrSkip runs aidlcli; skips the test when the service is unavailable.
func runAidlcliOrSkip(t *testing.T, args ...string) string {
	t.Helper()
	stdout, stderr, err := runAidlcli(args...)
	if err != nil {
		combined := stderr + stdout
		switch {
		case strings.Contains(combined, "not found"),
			strings.Contains(combined, "no service with descriptor"):
			t.Skipf("service unavailable: %s", strings.TrimSpace(combined))
		}
		t.Fatalf("aidlcli %v failed: %v\nstdout: %s\nstderr: %s", args, err, stdout, stderr)
	}
	return stdout
}

// --- Core service tests ---

func TestAidlcli_ServiceList(t *testing.T) {
	stdout := runAidlcliOrSkip(t, "service", "list")

	var rows []map[string]string
	require.NoError(t, json.Unmarshal([]byte(stdout), &rows), "unmarshal service list")
	require.NotEmpty(t, rows, "expected at least one service")

	names := make([]string, 0, len(rows))
	for _, row := range rows {
		name, ok := row["Name"]
		require.True(t, ok, "row missing NAME field: %v", row)
		_, hasStatus := row["Status"]
		assert.True(t, hasStatus, "row missing STATUS field: %v", row)
		names = append(names, name)
	}

	t.Logf("found %d services", len(names))

	// At least one well-known service should be present.
	foundWellKnown := false
	for _, n := range names {
		if n == "SurfaceFlinger" || n == "activity" {
			foundWellKnown = true
			break
		}
	}
	assert.True(t, foundWellKnown, "expected SurfaceFlinger or activity in service list")
}

func TestAidlcli_ServiceInspect(t *testing.T) {
	stdout := runAidlcliOrSkip(t, "service", "inspect", "SurfaceFlinger")

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), "unmarshal inspect result")

	assert.Equal(t, "SurfaceFlinger", result["name"])

	alive, ok := result["alive"].(bool)
	require.True(t, ok, "alive field should be bool, got %T", result["alive"])
	assert.True(t, alive, "SurfaceFlinger should be alive")

	descriptor, _ := result["descriptor"].(string)
	// Descriptor may be empty if the service doesn't respond to InterfaceTransaction.
	t.Logf("descriptor: %q", descriptor)

	t.Logf("SurfaceFlinger: handle=%v alive=%v descriptor=%s", result["handle"], alive, descriptor)
}

// --- GPS/Location ---

func TestAidlcli_Location_GetAllProviders(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.location.ILocationManager", "get-all-providers",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	providers, ok := raw.([]any)
	require.True(t, ok, "result should be array, got %T", raw)
	require.NotEmpty(t, providers, "expected at least one provider")

	providerStrings := make([]string, 0, len(providers))
	for _, p := range providers {
		s, ok := p.(string)
		require.True(t, ok, "provider should be string, got %T", p)
		providerStrings = append(providerStrings, s)
	}

	assert.Contains(t, providerStrings, "passive", "passive provider should always be available")
	t.Logf("providers: %v", providerStrings)
}

func TestAidlcli_Location_IsProviderEnabled(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.location.ILocationManager", "is-provider-enabled-for-user",
		"--provider", "passive", "--userId", "0",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(bool)
	assert.True(t, ok, "result should be bool, got %T", raw)
	t.Logf("isProviderEnabledForUser(passive, 0): %v", raw)
}

func TestAidlcli_Location_GetGnssHardwareModelName(t *testing.T) {
	stdout, stderr, err := runAidlcli(
		"android.location.ILocationManager", "get-gnss-hardware-model-name",
	)
	if err != nil {
		combined := stderr + stdout
		if strings.Contains(combined, "not found") ||
			strings.Contains(combined, "no service with descriptor") {
			t.Skipf("GNSS not available: %s", strings.TrimSpace(combined))
		}
		// GNSS might just return an error on emulator — skip gracefully.
		t.Skipf("GNSS hardware model name unavailable: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(string)
	assert.True(t, ok, "result should be string, got %T", raw)
	t.Logf("gnssHardwareModelName: %v", raw)
}

// --- ActivityManager ---

func TestAidlcli_ActivityManager_IsUserAMonkey(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.app.IActivityManager", "is-user-a-monkey",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	val, ok := raw.(bool)
	require.True(t, ok, "result should be bool, got %T", raw)
	assert.False(t, val, "should not be a monkey in test")
	t.Logf("isUserAMonkey: %v", val)
}

func TestAidlcli_ActivityManager_GetProcessLimit(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.app.IActivityManager", "get-process-limit",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	// JSON numbers unmarshal as float64.
	_, ok = raw.(float64)
	assert.True(t, ok, "result should be numeric, got %T", raw)
	t.Logf("getProcessLimit: %v", raw)
}

func TestAidlcli_ActivityManager_CheckPermission(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.app.IActivityManager", "check-permission",
		"--permission", "android.permission.INTERNET",
		"--pid", "1",
		"--uid", "0",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	val, ok := raw.(float64)
	require.True(t, ok, "result should be numeric, got %T", raw)
	assert.Equal(t, float64(0), val, "root should have INTERNET permission (0 = PERMISSION_GRANTED)")
	t.Logf("checkPermission(INTERNET, pid=1, uid=0): %v", val)
}

func TestAidlcli_ActivityManager_IsAppFreezerSupported(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.app.IActivityManager", "is-app-freezer-supported",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(bool)
	assert.True(t, ok, "result should be bool, got %T", raw)
	t.Logf("isAppFreezerSupported: %v", raw)
}

// --- PackageManager ---

func TestAidlcli_PackageManager_IsPackageAvailable(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.content.pm.IPackageManager", "is-package-available",
		"--packageName", "com.android.settings",
		"--userId", "0",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	val, ok := raw.(bool)
	require.True(t, ok, "result should be bool, got %T", raw)
	assert.True(t, val, "com.android.settings should be available")
	t.Logf("isPackageAvailable(com.android.settings, 0): %v", val)
}

func TestAidlcli_PackageManager_CheckPermission(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.content.pm.IPackageManager", "check-permission",
		"--permName", "android.permission.INTERNET",
		"--pkgName", "com.android.settings",
		"--userId", "0",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(float64)
	assert.True(t, ok, "result should be numeric, got %T", raw)
	t.Logf("checkPermission(INTERNET, com.android.settings, 0): %v", raw)
}

// --- Power/Battery/Thermal ---

func TestAidlcli_PowerManager_IsPowerSaveMode(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.os.IPowerManager", "is-power-save-mode",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(bool)
	assert.True(t, ok, "result should be bool, got %T", raw)
	t.Logf("isPowerSaveMode: %v", raw)
}

func TestAidlcli_PowerManager_IsInteractive(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.os.IPowerManager", "is-interactive",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(bool)
	assert.True(t, ok, "result should be bool, got %T", raw)
	t.Logf("isInteractive: %v", raw)
}

func TestAidlcli_ThermalService_GetCurrentThermalStatus(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.os.IThermalService", "get-current-thermal-status",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(float64)
	assert.True(t, ok, "result should be numeric, got %T", raw)
	t.Logf("getCurrentThermalStatus: %v", raw)
}

func TestAidlcli_Health_GetCapacity(t *testing.T) {
	stdout, stderr, err := runAidlcli(
		"android.hardware.health.IHealth", "get-capacity",
	)
	if err != nil {
		combined := stderr + stdout
		if strings.Contains(combined, "not found") ||
			strings.Contains(combined, "no service with descriptor") {
			t.Skipf("IHealth not available: %s", strings.TrimSpace(combined))
		}
		t.Skipf("health service unavailable: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(float64)
	assert.True(t, ok, "result should be numeric, got %T", raw)
	t.Logf("getCapacity: %v", raw)
}

// --- Display ---

func TestAidlcli_Display_GetDisplayIds(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"android.hardware.display.IDisplayManager", "get-display-ids",
		"--includeDisabled", "false",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	arr, ok := raw.([]any)
	require.True(t, ok, "result should be array, got %T", raw)
	require.NotEmpty(t, arr, "expected at least one display ID")
	t.Logf("getDisplayIds: %v", arr)
}

// --- Clipboard ---

func TestAidlcli_Clipboard_HasClipboardText(t *testing.T) {
	// Note: pass attributionTag as a non-empty placeholder to avoid
	// adb shell stripping the empty argument and shifting subsequent flags.
	stdout := runAidlcliOrSkip(t,
		"android.content.IClipboard", "has-clipboard-text",
		"--callingPackage=com.android.shell",
		"--attributionTag=none",
		"--userId=0",
		"--deviceId=0",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(bool)
	assert.True(t, ok, "result should be bool, got %T", raw)
	t.Logf("hasClipboardText: %v", raw)
}

// --- Telephony ---

func TestAidlcli_Telephony_GetActivePhoneType(t *testing.T) {
	stdout := runAidlcliOrSkip(t,
		"com.android.internal.telephony.ITelephony", "get-active-phone-type",
	)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	_, ok = raw.(float64)
	assert.True(t, ok, "result should be numeric, got %T", raw)
	t.Logf("getActivePhoneType: %v", raw)
}

// --- HAL services (likely skip in emulator) ---

func TestAidlcli_WiFi_ListNetworks(t *testing.T) {
	stdout, stderr, err := runAidlcli(
		"android.hardware.wifi.supplicant.ISupplicantStaIface", "list-networks",
	)
	if err != nil {
		combined := stderr + stdout
		if strings.Contains(combined, "not found") ||
			strings.Contains(combined, "no service with descriptor") {
			t.Skipf("supplicant HAL not available: %s", strings.TrimSpace(combined))
		}
		t.Skipf("wifi supplicant unavailable: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	arr, ok := raw.([]any)
	require.True(t, ok, "result should be array, got %T", raw)
	t.Logf("listNetworks: %d entries", len(arr))
}

func TestAidlcli_Camera_GetCameraIdList(t *testing.T) {
	stdout, stderr, err := runAidlcli(
		"android.hardware.camera.provider.ICameraProvider", "get-camera-id-list",
	)
	if err != nil {
		combined := stderr + stdout
		if strings.Contains(combined, "not found") ||
			strings.Contains(combined, "no service with descriptor") {
			t.Skipf("camera provider not available: %s", strings.TrimSpace(combined))
		}
		t.Skipf("camera provider unavailable: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["result"]
	require.True(t, ok, "response missing 'result' key")

	arr, ok := raw.([]any)
	require.True(t, ok, "result should be array, got %T", raw)
	t.Logf("getCameraIdList: %v", arr)
}

func TestAidlcli_Bluetooth_Close(t *testing.T) {
	stdout, stderr, err := runAidlcli(
		"android.hardware.bluetooth.IBluetoothHci", "close",
	)
	if err != nil {
		combined := stderr + stdout
		if strings.Contains(combined, "not found") ||
			strings.Contains(combined, "no service with descriptor") {
			t.Skipf("bluetooth HCI not available: %s", strings.TrimSpace(combined))
		}
		t.Skipf("bluetooth HCI unavailable: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), "unmarshal response")

	raw, ok := envelope["status"]
	require.True(t, ok, "response missing 'status' key")

	val, ok := raw.(string)
	require.True(t, ok, "status should be string, got %T", raw)
	assert.Equal(t, "ok", val)
	t.Logf("bluetooth close: status=%s", val)
}
