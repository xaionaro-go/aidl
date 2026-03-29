// Query power save mode status and related settings via PowerManager.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/power_save_auto ./examples/power_save_auto/
//	adb push build/power_save_auto /data/local/tmp/ && adb shell /data/local/tmp/power_save_auto
package main

import (
	"context"
	"fmt"
	"os"

	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func main() {
	ctx := context.Background()

	drv, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer drv.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)

	power, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get power service: %v\n", err)
		os.Exit(1)
	}

	powerSave, err := power.IsPowerSaveMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsPowerSaveMode: %v\n", err)
	} else {
		fmt.Printf("Power save mode:          %v\n", powerSave)
	}

	autoModes, err := power.AreAutoPowerSaveModesEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "AreAutoPowerSaveModesEnabled: %v\n", err)
	} else {
		fmt.Printf("Auto power save enabled:  %v\n", autoModes)
	}

	batterySaver, err := power.IsBatterySaverSupported(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsBatterySaverSupported: %v\n", err)
	} else {
		fmt.Printf("Battery saver supported:  %v\n", batterySaver)
	}

	trigger, err := power.GetPowerSaveModeTrigger(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetPowerSaveModeTrigger: %v\n", err)
	} else {
		fmt.Printf("Power save trigger:       %d\n", trigger)
	}

	idle, err := power.IsDeviceIdleMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsDeviceIdleMode: %v\n", err)
	} else {
		fmt.Printf("Device idle (Doze):       %v\n", idle)
	}

	lightIdle, err := power.IsLightDeviceIdleMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsLightDeviceIdleMode: %v\n", err)
	} else {
		fmt.Printf("Light device idle:        %v\n", lightIdle)
	}

	lowPower, err := power.IsLowPowerStandbyEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsLowPowerStandbyEnabled: %v\n", err)
	} else {
		fmt.Printf("Low power standby:        %v\n", lowPower)
	}

	lowPowerSupported, err := power.IsLowPowerStandbySupported(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsLowPowerStandbySupported: %v\n", err)
	} else {
		fmt.Printf("Low power standby support: %v\n", lowPowerSupported)
	}
}
