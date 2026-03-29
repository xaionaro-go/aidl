//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/AndroidGoLab/binder/android/hardware/health"
	"github.com/AndroidGoLab/binder/android/location"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/android/system/suspend"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

// -----------------------------------------------------------------------
// Use case #11: power_profiling — Battery current draw measurement
// -----------------------------------------------------------------------

func TestUseCase11_PowerProfiling(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	svc, err := sm.GetService(ctx, servicemanager.ServiceName(health.DescriptorIHealth+"/default"))
	requireOrSkip(t, err)

	h := health.NewHealthProxy(svc)

	now, err := h.GetCurrentNowMicroamps(ctx)
	requireOrSkip(t, err)
	t.Logf("CurrentNowMicroamps: %d uA (%.1f mA)", now, float64(now)/1000.0)

	avg, err := h.GetCurrentAverageMicroamps(ctx)
	requireOrSkip(t, err)
	t.Logf("CurrentAverageMicroamps: %d uA (%.1f mA)", avg, float64(avg)/1000.0)

	counter, err := h.GetChargeCounterUah(ctx)
	requireOrSkip(t, err)
	t.Logf("ChargeCounterUah: %d", counter)

	energy, err := h.GetEnergyCounterNwh(ctx)
	requireOrSkip(t, err)
	t.Logf("EnergyCounterNwh: %d", energy)
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
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	svc, err := sm.GetService(ctx, servicemanager.ServiceName(suspend.DescriptorISystemSuspend+"/default"))
	requireOrSkip(t, err)

	ss := suspend.NewSystemSuspendProxy(svc)

	// Acquire a partial wake lock.
	wl, err := ss.AcquireWakeLock(ctx, suspend.WakeLockTypePARTIAL, "e2e_test_suspend")
	requireOrSkip(t, err)
	t.Logf("Acquired wake lock via SystemSuspend")

	// Release the wake lock.
	err = wl.Release(ctx)
	requireOrSkip(t, err)
	t.Logf("Released wake lock via SystemSuspend")
}

// -----------------------------------------------------------------------
// Use case #16: charge_monitor — Monitor charging status via Health HAL
// -----------------------------------------------------------------------

func TestUseCase16_ChargeMonitor(t *testing.T) {
	ctx := context.Background()
	driver := openBinder(t)
	sm := servicemanager.New(driver)

	svc, err := sm.GetService(ctx, servicemanager.ServiceName(health.DescriptorIHealth+"/default"))
	requireOrSkip(t, err)

	h := health.NewHealthProxy(svc)

	status, err := h.GetChargeStatus(ctx)
	requireOrSkip(t, err)
	t.Logf("ChargeStatus: %d", status)
	// Status should be one of the known BatteryStatus values (1-5).
	assert.True(t, status >= 1 && status <= 5,
		"charge status should be between 1 and 5, got %d", status)

	capacity, err := h.GetCapacity(ctx)
	requireOrSkip(t, err)
	t.Logf("Battery capacity: %d%%", capacity)
	assert.True(t, capacity >= 0 && capacity <= 100,
		"capacity should be 0-100, got %d", capacity)

	info, err := h.GetHealthInfo(ctx)
	requireOrSkip(t, err)
	t.Logf("HealthInfo: AC=%v USB=%v Wireless=%v Present=%v Voltage=%dmV Temp=%.1fC",
		info.ChargerAcOnline, info.ChargerUsbOnline, info.ChargerWirelessOnline,
		info.BatteryPresent, info.BatteryVoltageMillivolts,
		float64(info.BatteryTemperatureTenthsCelsius)/10.0)

	policy, err := h.GetChargingPolicy(ctx)
	requireOrSkip(t, err)
	t.Logf("ChargingPolicy: %d", policy)

	healthData, err := h.GetBatteryHealthData(ctx)
	requireOrSkip(t, err)
	t.Logf("BatteryHealthData: %+v", healthData)
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
