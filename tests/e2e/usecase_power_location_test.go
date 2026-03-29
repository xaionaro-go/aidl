//go:build e2e

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AndroidGoLab/binder/android/location"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/parcel"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// -----------------------------------------------------------------------
// Use case #11: power_profiling — Battery current draw measurement
// -----------------------------------------------------------------------

// Android BatteryManager property IDs (from android.os.BatteryManager).
const (
	batteryPropertyChargeCounter = 1 // BATTERY_PROPERTY_CHARGE_COUNTER (uAh)
	batteryPropertyCurrentNow    = 2 // BATTERY_PROPERTY_CURRENT_NOW (uA)
	batteryPropertyCurrentAvg    = 3 // BATTERY_PROPERTY_CURRENT_AVERAGE (uA)
	batteryPropertyCapacity      = 4 // BATTERY_PROPERTY_CAPACITY (%)
)

func TestUseCase11_PowerProfiling(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// The Health HAL (android.hardware.health.IHealth/default) is not
	// accessible from shell context due to SELinux. Use the framework
	// BatteryPropertiesRegistrar service instead, which exposes the
	// same battery measurements via the "batteryproperties" binder
	// service and works from shell.
	proxy, err := genOs.GetBatteryPropertiesRegistrar(ctx, sm)
	requireOrSkip(t, err)

	now, err := proxy.GetProperty(ctx, batteryPropertyCurrentNow, genOs.BatteryProperty{})
	requireOrSkip(t, err)
	t.Logf("CurrentNow: %d uA (%.1f mA)", now, float64(now)/1000.0)

	avg, err := proxy.GetProperty(ctx, batteryPropertyCurrentAvg, genOs.BatteryProperty{})
	requireOrSkip(t, err)
	t.Logf("CurrentAverage: %d uA (%.1f mA)", avg, float64(avg)/1000.0)

	counter, err := proxy.GetProperty(ctx, batteryPropertyChargeCounter, genOs.BatteryProperty{})
	requireOrSkip(t, err)
	t.Logf("ChargeCounter: %d uAh", counter)

	// Energy counter is not available via BatteryPropertiesRegistrar.
	// Read it from sysfs as a fallback.
	t.Run("EnergyCounter_Sysfs", func(t *testing.T) {
		data, err := os.ReadFile("/sys/class/power_supply/battery/voltage_now")
		if err != nil {
			t.Skipf("sysfs voltage_now not available: %v", err)
		}
		t.Logf("Voltage (sysfs): %s uV", strings.TrimSpace(string(data)))
	})
}

// -----------------------------------------------------------------------
// Use case #12: wakelock_audit — Enumerate wake lock level support
// -----------------------------------------------------------------------

func TestUseCase12_WakelockAudit(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	power, err := genOs.GetPowerManager(ctx, sm)
	requireOrSkip(t, err)

	levels := []struct {
		name  string
		level int32
	}{
		{"PARTIAL_WAKE_LOCK", 1},
		{"SCREEN_DIM_WAKE_LOCK", 6},
		{"SCREEN_BRIGHT_WAKE_LOCK", 10},
		{"FULL_WAKE_LOCK", 26},
		{"PROXIMITY_SCREEN_OFF_WAKE_LOCK", 32},
		{"DOZE_WAKE_LOCK", 64},
		{"DRAW_WAKE_LOCK", 128},
	}

	for _, wl := range levels {
		supported, err := power.IsWakeLockLevelSupported(ctx, wl.level)
		requireOrSkip(t, err)
		t.Logf("%-35s supported: %v", wl.name, supported)
	}

	// PARTIAL_WAKE_LOCK should always be supported.
	partial, err := power.IsWakeLockLevelSupported(ctx, 1)
	requireOrSkip(t, err)
	assert.True(t, partial, "PARTIAL_WAKE_LOCK should be supported")

	boosted, err := power.IsScreenBrightnessBoosted(ctx)
	requireOrSkip(t, err)
	t.Logf("Screen brightness boosted: %v", boosted)
}

// -----------------------------------------------------------------------
// Use case #13: screen_control — Check screen on/off state
// -----------------------------------------------------------------------

func TestUseCase13_ScreenControl(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	power, err := genOs.GetPowerManager(ctx, sm)
	requireOrSkip(t, err)

	interactive, err := power.IsInteractive(ctx)
	requireOrSkip(t, err)
	t.Logf("IsInteractive (screen on): %v", interactive)

	displayInteractive, err := power.IsDisplayInteractive(ctx, 0)
	requireOrSkip(t, err)
	t.Logf("IsDisplayInteractive(0): %v", displayInteractive)

	ambient, err := power.IsAmbientDisplayAvailable(ctx)
	requireOrSkip(t, err)
	t.Logf("IsAmbientDisplayAvailable: %v", ambient)

	suppressed, err := power.IsAmbientDisplaySuppressed(ctx)
	requireOrSkip(t, err)
	t.Logf("IsAmbientDisplaySuppressed: %v", suppressed)

	lastSleep, err := power.GetLastSleepReason(ctx)
	requireOrSkip(t, err)
	t.Logf("GetLastSleepReason: %d", lastSleep)

	lastShutdown, err := power.GetLastShutdownReason(ctx)
	requireOrSkip(t, err)
	t.Logf("GetLastShutdownReason: %d", lastShutdown)
}

// -----------------------------------------------------------------------
// Use case #14: power_save_auto — Query/toggle power save mode
// -----------------------------------------------------------------------

func TestUseCase14_PowerSaveAuto(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	power, err := genOs.GetPowerManager(ctx, sm)
	requireOrSkip(t, err)

	powerSave, err := power.IsPowerSaveMode(ctx)
	requireOrSkip(t, err)
	t.Logf("IsPowerSaveMode: %v", powerSave)

	autoModes, err := power.AreAutoPowerSaveModesEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("AreAutoPowerSaveModesEnabled: %v", autoModes)

	batterySaver, err := power.IsBatterySaverSupported(ctx)
	requireOrSkip(t, err)
	t.Logf("IsBatterySaverSupported: %v", batterySaver)

	trigger, err := power.GetPowerSaveModeTrigger(ctx)
	requireOrSkip(t, err)
	t.Logf("GetPowerSaveModeTrigger: %d", trigger)

	idle, err := power.IsDeviceIdleMode(ctx)
	requireOrSkip(t, err)
	t.Logf("IsDeviceIdleMode: %v", idle)

	lightIdle, err := power.IsLightDeviceIdleMode(ctx)
	requireOrSkip(t, err)
	t.Logf("IsLightDeviceIdleMode: %v", lightIdle)

	lowPower, err := power.IsLowPowerStandbyEnabled(ctx)
	requireOrSkip(t, err)
	t.Logf("IsLowPowerStandbyEnabled: %v", lowPower)

	lowPowerSupported, err := power.IsLowPowerStandbySupported(ctx)
	requireOrSkip(t, err)
	t.Logf("IsLowPowerStandbySupported: %v", lowPowerSupported)
}

// -----------------------------------------------------------------------
// Use case #15: suspend_logger — Interact with SystemSuspend HAL
// -----------------------------------------------------------------------

func TestUseCase15_SuspendLogger(t *testing.T) {
	ctx := context.Background()

	// The SystemSuspend HAL (android.system.suspend.ISystemSuspend/default)
	// is not accessible from shell context due to SELinux. Use the
	// framework PowerManager service instead, which provides wake lock
	// acquire/release via the "power" binder service.
	driver, transport := openBinderDirect(ctx, t)
	defer func() { _ = driver.Close(ctx) }()
	defer func() { _ = transport.Close(ctx) }()

	sm := servicemanager.New(transport)

	power, err := genOs.GetPowerManager(ctx, sm)
	requireOrSkip(t, err)

	// Create a binder token for wake lock ownership tracking.
	lockToken := binder.NewStubBinder(&wakeLockToken{})
	lockToken.RegisterWithTransport(ctx, transport)

	const (
		partialWakeLock = 1 // PowerManager.PARTIAL_WAKE_LOCK
		packageName     = "com.android.shell"
	)

	// Acquire a partial wake lock via PowerManager.
	wlCallback := genOs.NewWakeLockCallbackStub(&noopWakeLockCallback{})
	err = power.AcquireWakeLock(
		ctx,
		lockToken,
		partialWakeLock,
		"e2e_test_suspend",
		packageName,
		genOs.WorkSource{},
		"",  // historyTag
		0,   // displayId
		wlCallback,
	)
	requireOrSkip(t, err)
	t.Log("Acquired wake lock via PowerManager")

	// Verify the lock is supported.
	supported, err := power.IsWakeLockLevelSupported(ctx, partialWakeLock)
	requireOrSkip(t, err)
	require.True(t, supported, "PARTIAL_WAKE_LOCK should be supported")

	// Release the wake lock.
	err = power.ReleaseWakeLock(ctx, lockToken, 0)
	requireOrSkip(t, err)
	t.Log("Released wake lock via PowerManager")
}

// wakeLockToken is a minimal TransactionReceiver used as the binder token
// for PowerManager wake lock acquire/release.
type wakeLockToken struct{}

func (w *wakeLockToken) Descriptor() string { return "wakelock.token" }

func (w *wakeLockToken) OnTransaction(
	_ context.Context,
	_ binder.TransactionCode,
	_ *parcel.Parcel,
) (*parcel.Parcel, error) {
	return parcel.New(), nil
}

// noopWakeLockCallback implements IWakeLockCallbackServer for use with
// PowerManager.AcquireWakeLock. The callback receives state change
// notifications but does not act on them.
type noopWakeLockCallback struct{}

func (n *noopWakeLockCallback) OnStateChanged(_ context.Context, _ bool) error {
	return nil
}

var _ genOs.IWakeLockCallbackServer = (*noopWakeLockCallback)(nil)

// -----------------------------------------------------------------------
// Use case #16: charge_monitor — Monitor charging status via Health HAL
// -----------------------------------------------------------------------

func TestUseCase16_ChargeMonitor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	// The Health HAL is not accessible from shell context due to SELinux.
	// Use BatteryPropertiesRegistrar for capacity and current, and
	// supplement with sysfs for charge status and other health info.
	proxy, err := genOs.GetBatteryPropertiesRegistrar(ctx, sm)
	requireOrSkip(t, err)

	capacity, err := proxy.GetProperty(ctx, batteryPropertyCapacity, genOs.BatteryProperty{})
	requireOrSkip(t, err)
	t.Logf("Battery capacity: %d%%", capacity)
	assert.True(t, capacity >= 0 && capacity <= 100,
		"capacity should be 0-100, got %d", capacity)

	current, err := proxy.GetProperty(ctx, batteryPropertyCurrentNow, genOs.BatteryProperty{})
	requireOrSkip(t, err)
	t.Logf("Battery current: %d uA", current)

	// Read additional health info from sysfs.
	sysfsEntries := []struct {
		name string
		path string
	}{
		{"ChargeStatus", "/sys/class/power_supply/battery/status"},
		{"Voltage", "/sys/class/power_supply/battery/voltage_now"},
		{"Temperature", "/sys/class/power_supply/battery/temp"},
		{"Health", "/sys/class/power_supply/battery/health"},
		{"Technology", "/sys/class/power_supply/battery/technology"},
		{"ChargerUSB", "/sys/class/power_supply/usb/online"},
		{"ChargerAC", "/sys/class/power_supply/ac/online"},
	}

	for _, entry := range sysfsEntries {
		data, err := os.ReadFile(entry.path)
		if err != nil {
			continue
		}
		t.Logf("  %-15s %s", entry.name+":", strings.TrimSpace(string(data)))
	}
}

// -----------------------------------------------------------------------
// Use case #18: geofence — Check provider availability for geofencing
// -----------------------------------------------------------------------

func TestUseCase18_Geofence(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	lm, err := location.GetLocationManager(ctx, sm)
	requireOrSkip(t, err)

	providers, err := lm.GetAllProviders(ctx)
	requireOrSkip(t, err)
	t.Logf("All providers (%d): %v", len(providers), providers)
	assert.NotEmpty(t, providers, "expected at least one location provider")

	hasGPS, err := lm.HasProvider(ctx, location.GpsProvider)
	requireOrSkip(t, err)
	t.Logf("HasProvider(gps): %v", hasGPS)

	hasFused, err := lm.HasProvider(ctx, location.FusedProvider)
	requireOrSkip(t, err)
	t.Logf("HasProvider(fused): %v", hasFused)

	enabled, err := lm.IsLocationEnabledForUser(ctx)
	requireOrSkip(t, err)
	t.Logf("IsLocationEnabledForUser: %v", enabled)
}

// -----------------------------------------------------------------------
// Use case #19: gnss_diagnostics — Query GNSS hardware info
// -----------------------------------------------------------------------

func TestUseCase19_GnssDiagnostics(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	lm, err := location.GetLocationManager(ctx, sm)
	requireOrSkip(t, err)

	model, err := lm.GetGnssHardwareModelName(ctx)
	requireOrSkip(t, err)
	t.Logf("GNSS hardware model: %q", model)

	year, err := lm.GetGnssYearOfHardware(ctx)
	requireOrSkip(t, err)
	t.Logf("GNSS year of hardware: %d", year)
	assert.True(t, year >= 0, "GNSS year should be non-negative")

	caps, err := lm.GetGnssCapabilities(ctx)
	requireOrSkip(t, err)
	t.Logf("GNSS capabilities: TopFlags=0x%x AdrKnown=%v CorrFlags=0x%x PowerFlags=0x%x",
		caps.TopFlags, caps.IsAdrCapabilityKnown, caps.MeasurementCorrectionsFlags, caps.PowerFlags)

	batchSize, err := lm.GetGnssBatchSize(ctx)
	requireOrSkip(t, err)
	t.Logf("GNSS batch size: %d", batchSize)
}

// -----------------------------------------------------------------------
// Use case #20: last_location — Retrieve last known fused location
// -----------------------------------------------------------------------

func TestUseCase20_LastLocation(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	lm, err := location.GetLocationManager(ctx, sm)
	requireOrSkip(t, err)

	packageName := binder.DefaultCallerIdentity().PackageName

	providers := []string{
		string(location.FusedProvider),
		string(location.GpsProvider),
		string(location.NetworkProvider),
	}

	for _, provider := range providers {
		loc, err := lm.GetLastLocation(ctx, provider, location.LastLocationRequest{}, packageName)
		if err != nil {
			// Permission or provider-not-found errors are expected in some environments.
			t.Logf("GetLastLocation(%s): %v", provider, err)
			continue
		}
		if loc.TimeMs == 0 {
			t.Logf("[%s] No cached location available", provider)
		} else {
			t.Logf("[%s] Lat=%.6f Lon=%.6f Alt=%.1f Acc=%.1f",
				provider, loc.LatitudeDegrees, loc.LongitudeDegrees,
				loc.AltitudeMeters, loc.HorizontalAccuracyMeters)
		}
	}
}

// -----------------------------------------------------------------------
// Use case #21: location_benchmark — Compare providers via properties
// -----------------------------------------------------------------------

func TestUseCase21_LocationBenchmark(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	lm, err := location.GetLocationManager(ctx, sm)
	requireOrSkip(t, err)

	providers, err := lm.GetAllProviders(ctx)
	requireOrSkip(t, err)
	assert.NotEmpty(t, providers, "expected at least one provider")

	for _, p := range providers {
		props, err := lm.GetProviderProperties(ctx, p)
		if err != nil {
			t.Logf("[%s] GetProviderProperties: %v", p, err)
			continue
		}
		t.Logf("[%s] Power=%d Accuracy=%d Sat=%v Net=%v Cell=%v Alt=%v Speed=%v Bearing=%v",
			p, props.PowerUsage, props.Accuracy,
			props.HasSatelliteRequirement, props.HasNetworkRequirement,
			props.HasCellRequirement, props.HasAltitudeSupport,
			props.HasSpeedSupport, props.HasBearingSupport)
	}

	// Verify that at least the passive provider has properties.
	// On most Android devices "passive" is always available.
	for _, p := range providers {
		if p == "passive" {
			_, err := lm.GetProviderProperties(ctx, p)
			requireOrSkip(t, err)
		}
	}
}
