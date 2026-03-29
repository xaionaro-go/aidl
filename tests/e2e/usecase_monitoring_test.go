//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	genApp "github.com/AndroidGoLab/binder/android/app"
	"github.com/AndroidGoLab/binder/android/hardware/display"
	"github.com/AndroidGoLab/binder/android/hardware/usb"
	"github.com/AndroidGoLab/binder/android/media"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/android/os/storage"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// ==========================================================================
// Use case #1: Battery health
// ==========================================================================

func TestUsecase_BatteryHealth(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	proxy, err := genOs.GetBatteryPropertiesRegistrar(ctx, sm)
	requireOrSkip(t, err)

	// Query battery capacity (%).
	const batteryPropertyCapacity = 4
	cap, err := proxy.GetProperty(ctx, batteryPropertyCapacity, genOs.BatteryProperty{})
	requireOrSkip(t, err)
	t.Logf("Battery capacity: %d%%", cap)
	require.GreaterOrEqual(t, int32(cap), int32(0), "capacity should be >= 0")
	require.LessOrEqual(t, int32(cap), int32(100), "capacity should be <= 100")

	// Query current draw.
	const batteryPropertyCurrentNow = 2
	current, err := proxy.GetProperty(ctx, batteryPropertyCurrentNow, genOs.BatteryProperty{})
	requireOrSkip(t, err)
	t.Logf("Battery current: %d uA", current)
}

// ==========================================================================
// Use case #2: Thermal throttling monitor
// ==========================================================================

func TestUsecase_ThermalMonitor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "thermalservice")

	thermal := genOs.NewThermalServiceProxy(svc)

	// Thermal status (0=none .. 6=shutdown).
	status, err := thermal.GetCurrentThermalStatus(ctx)
	requireOrSkip(t, err)
	t.Logf("Thermal status: %d", status)
	require.GreaterOrEqual(t, status, int32(0))
	require.LessOrEqual(t, status, int32(6))

	// Thermal headroom forecast.
	headroom, err := thermal.GetThermalHeadroom(ctx, 10)
	requireOrSkip(t, err)
	t.Logf("Thermal headroom (10s): %.2f", headroom)

	// Temperature sensors.
	temps, err := thermal.GetCurrentTemperatures(ctx)
	requireOrSkip(t, err)
	t.Logf("Temperature sensors: %d", len(temps))
	for i, tmp := range temps {
		if i >= 5 {
			t.Logf("  ... and %d more", len(temps)-5)
			break
		}
		t.Logf("  %s: %.1f C (type=%d status=%d)", tmp.Name, tmp.Value, tmp.Type, tmp.Status)
	}

	// Cooling devices.
	coolers, err := thermal.GetCurrentCoolingDevices(ctx)
	requireOrSkip(t, err)
	t.Logf("Cooling devices: %d", len(coolers))
	for i, c := range coolers {
		if i >= 5 {
			t.Logf("  ... and %d more", len(coolers)-5)
			break
		}
		t.Logf("  %s: value=%d type=%d", c.Name, c.Value, c.Type)
	}
}

func TestUsecase_ThermalMonitor_HardwareProperties(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	hwProps, err := genOs.GetHardwarePropertiesManager(ctx, sm)
	requireOrSkip(t, err)

	// CPU temperatures.
	cpuTemps, err := hwProps.GetDeviceTemperatures(ctx, 0, 0)
	requireOrSkip(t, err)
	t.Logf("CPU temperatures: %v", cpuTemps)

	// CPU usage per core.
	cpuUsages, err := hwProps.GetCpuUsages(ctx)
	requireOrSkip(t, err)
	t.Logf("CPU cores: %d", len(cpuUsages))
	for i, u := range cpuUsages {
		if i >= 4 {
			t.Logf("  ... and %d more cores", len(cpuUsages)-4)
			break
		}
		t.Logf("  Core %d: active=%d total=%d", i, u.Active, u.Total)
	}

	// Fan speeds.
	fans, err := hwProps.GetFanSpeeds(ctx)
	requireOrSkip(t, err)
	t.Logf("Fan speeds: %v", fans)
}

// ==========================================================================
// Use case #3: Process watchdog
// ==========================================================================

func TestUsecase_ProcessWatchdog(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "activity")

	am := genApp.NewActivityManagerProxy(svc)

	// Process limit.
	limit, err := am.GetProcessLimit(ctx)
	requireOrSkip(t, err)
	t.Logf("Process limit: %d", limit)

	// Monkey mode.
	monkey, err := am.IsUserAMonkey(ctx)
	requireOrSkip(t, err)
	t.Logf("Monkey mode: %v", monkey)

	// App freezer.
	freezer, err := am.IsAppFreezerSupported(ctx)
	requireOrSkip(t, err)
	t.Logf("App freezer supported: %v", freezer)

	// Running processes.
	procs, err := am.GetRunningAppProcesses(ctx)
	requireOrSkip(t, err)
	t.Logf("Running processes: %d", len(procs))

	// Lock task mode.
	lockMode, err := am.GetLockTaskModeState(ctx)
	requireOrSkip(t, err)
	t.Logf("Lock task mode: %d", lockMode)

	// UID activity check (root UID = 0).
	active, err := am.IsUidActive(ctx, 0)
	requireOrSkip(t, err)
	t.Logf("UID 0 active: %v", active)

	// UID process state.
	procState, err := am.GetUidProcessState(ctx, 0)
	requireOrSkip(t, err)
	t.Logf("UID 0 process state: %d", procState)
}

// ==========================================================================
// Use case #4: Storage usage
// ==========================================================================

func TestUsecase_StorageUsage(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "mount")

	store := storage.NewStorageManagerProxy(svc)

	lastMaint, err := store.LastMaintenance(ctx)
	requireOrSkip(t, err)
	t.Logf("Last maintenance (fstrim): %d ms since epoch", lastMaint)
	require.Greater(t, lastMaint, int64(0), "last maintenance should be positive")
}

// ==========================================================================
// Use case #5: Memory pressure monitor
// ==========================================================================

func TestUsecase_MemoryPressure(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "activity")

	am := genApp.NewActivityManagerProxy(svc)

	// Memory info (struct fields not exposed, but call must succeed).
	memInfo := genApp.ActivityManagerMemoryInfo{}
	err := am.GetMemoryInfo(ctx, memInfo)
	requireOrSkip(t, err)
	t.Log("GetMemoryInfo: succeeded")

	// Memory trim level.
	trimLevel, err := am.GetMemoryTrimLevel(ctx)
	requireOrSkip(t, err)
	t.Logf("Memory trim level: %d", trimLevel)

	// Own memory state.
	myState := genApp.ActivityManagerRunningAppProcessInfo{}
	err = am.GetMyMemoryState(ctx, myState)
	requireOrSkip(t, err)
	t.Log("GetMyMemoryState: succeeded")

	// Process memory info for PID 1 (init).
	memInfos, err := am.GetProcessMemoryInfo(ctx, []int32{1})
	requireOrSkip(t, err)
	t.Logf("GetProcessMemoryInfo(pid=1): %d entries", len(memInfos))
}

// ==========================================================================
// Use case #6: Device info
// ==========================================================================

func TestUsecase_DeviceInfo(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)

	// Thermal status (part of device info).
	thermalSvc := getService(ctx, t, driver, "thermalservice")
	thermal := genOs.NewThermalServiceProxy(thermalSvc)

	status, err := thermal.GetCurrentThermalStatus(ctx)
	requireOrSkip(t, err)
	t.Logf("Thermal status: %d", status)

	headroom, err := thermal.GetThermalHeadroom(ctx, 10)
	requireOrSkip(t, err)
	t.Logf("Thermal headroom: %.2f", headroom)

	// Network interfaces.
	netSvc := getService(ctx, t, driver, "network_management")
	net := genOs.NewNetworkManagementServiceProxy(netSvc)

	ifaces, err := net.ListInterfaces(ctx)
	requireOrSkip(t, err)
	t.Logf("Network interfaces: %v", ifaces)
	require.NotEmpty(t, ifaces, "device should have at least one network interface")

	bw, err := net.IsBandwidthControlEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("Bandwidth control: %v", bw)
}

// ==========================================================================
// Use case #7: Display diagnostics
// ==========================================================================

func TestUsecase_DisplayDiagnostics(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "display")

	dm := display.NewDisplayManagerProxy(svc)

	ids, err := dm.GetDisplayIds(ctx, false)
	requireOrSkip(t, err)
	t.Logf("Display IDs: %v", ids)
	require.NotEmpty(t, ids, "device should have at least one display")

	for _, id := range ids {
		brightness, err := dm.GetBrightness(ctx, id)
		requireOrSkip(t, err)
		t.Logf("Display %d brightness: %.2f", id, brightness)
	}

	// Color display.
	colorSvc := getService(ctx, t, driver, "color_display")
	cdm := display.NewColorDisplayManagerProxy(colorSvc)

	nightMode, err := cdm.IsNightDisplayActivated(ctx)
	requireOrSkip(t, err)
	t.Logf("Night mode: %v", nightMode)

	colorMode, err := cdm.GetColorMode(ctx)
	requireOrSkip(t, err)
	t.Logf("Color mode: %d", colorMode)

	managed, err := cdm.IsDeviceColorManaged(ctx)
	requireOrSkip(t, err)
	t.Logf("Color managed: %v", managed)
}

// ==========================================================================
// Use case #8: Audio device enumerator
// ==========================================================================

func TestUsecase_AudioStatus(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	svc := getService(ctx, t, driver, "audio")

	audio := media.NewAudioServiceProxy(svc)

	mode, err := audio.GetMode(ctx)
	requireOrSkip(t, err)
	t.Logf("Audio mode: %d", mode)

	micMuted, err := audio.IsMicrophoneMuted(ctx)
	requireOrSkip(t, err)
	t.Logf("Mic muted: %v", micMuted)

	// Volume levels for common streams.
	for stream := int32(0); stream <= 5; stream++ {
		vol, err := audio.GetStreamVolume(ctx, stream)
		if err != nil {
			continue
		}
		maxVol, err := audio.GetStreamMaxVolume(ctx, stream)
		if err != nil {
			continue
		}
		t.Logf("Stream %d: volume=%d/%d", stream, vol, maxVol)
	}
}

// ==========================================================================
// Use case #9: Network connectivity monitor
// ==========================================================================

func TestUsecase_NetworkMonitor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// Connectivity service is Java-only AIDL, so we can only ping it.
	connSvc, err := sm.CheckService(ctx, servicemanager.ConnectivityService)
	requireOrSkip(t, err)
	if connSvc == nil {
		t.Skip("connectivity service not registered")
	}
	require.True(t, connSvc.IsAlive(ctx), "connectivity service should be alive")
	t.Logf("Connectivity service: alive, handle=%d", connSvc.Handle())

	// Network management service has full AIDL support.
	netSvc := getService(ctx, t, driver, "network_management")
	net := genOs.NewNetworkManagementServiceProxy(netSvc)

	// Interface list.
	ifaces, err := net.ListInterfaces(ctx)
	requireOrSkip(t, err)
	t.Logf("Network interfaces: %v", ifaces)
	require.NotEmpty(t, ifaces)

	// Each method tested independently — some may be removed in newer
	// API levels (e.g. API 36 removed getIpForwardingEnabled,
	// isTetheringStarted).
	t.Run("IpForwarding", func(t *testing.T) {
		fwd, err := net.GetIpForwardingEnabled(ctx)
		requireOrSkip(t, err)
		t.Logf("IP forwarding: %v", fwd)
	})

	t.Run("BandwidthControl", func(t *testing.T) {
		bw, err := net.IsBandwidthControlEnabled(ctx)
		requireOrSkip(t, err)
		t.Logf("Bandwidth control: %v", bw)
	})

	t.Run("Firewall", func(t *testing.T) {
		fw, err := net.IsFirewallEnabled(ctx)
		requireOrSkip(t, err)
		t.Logf("Firewall: %v", fw)
	})

	t.Run("Tethering", func(t *testing.T) {
		teth, err := net.IsTetheringStarted(ctx)
		requireOrSkip(t, err)
		t.Logf("Tethering: %v", teth)
	})

	t.Run("NetworkRestriction", func(t *testing.T) {
		restricted, err := net.IsNetworkRestricted(ctx, 0)
		requireOrSkip(t, err)
		t.Logf("UID 0 restricted: %v", restricted)
	})
}

// ==========================================================================
// Use case #10: USB device tracker
// ==========================================================================

func TestUsecase_UsbTracker(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	usbMgr, err := usb.GetUsbManager(ctx, sm)
	requireOrSkip(t, err)

	// Current functions.
	funcs, err := usbMgr.GetCurrentFunctions(ctx)
	requireOrSkip(t, err)
	t.Logf("USB functions: 0x%x", funcs)

	// USB speed.
	speed, err := usbMgr.GetCurrentUsbSpeed(ctx)
	requireOrSkip(t, err)
	t.Logf("USB speed: %d", speed)

	// Gadget HAL version.
	gadgetVer, err := usbMgr.GetGadgetHalVersion(ctx)
	requireOrSkip(t, err)
	t.Logf("Gadget HAL version: %d", gadgetVer)

	// USB HAL version.
	usbHalVer, err := usbMgr.GetUsbHalVersion(ctx)
	requireOrSkip(t, err)
	t.Logf("USB HAL version: %d", usbHalVer)

	// USB ports.
	ports, err := usbMgr.GetPorts(ctx)
	requireOrSkip(t, err)
	t.Logf("USB ports: %d", len(ports))
	for _, p := range ports {
		t.Logf("  Port %q: modes=%d contamProtect=%d altModes=0x%x",
			p.Id, p.SupportedModes, p.SupportedContaminantProtectionModes, p.SupportedAltModesMask)
	}

	// Screen-unlocked functions.
	screenFuncs, err := usbMgr.GetScreenUnlockedFunctions(ctx)
	requireOrSkip(t, err)
	t.Logf("Screen-unlocked functions: 0x%x", screenFuncs)
}
