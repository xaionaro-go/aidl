// Monitor user presence via PowerManager and display state.
//
// The AttentionService requires a callback binder and system permissions,
// making it inaccessible from shell context. This example achieves
// similar user-presence detection by querying PowerManager (IsInteractive,
// IsDeviceIdleMode, IsPowerSaveMode) and DisplayManager (brightness),
// which together indicate whether a user is actively using the device.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/attention_monitor ./examples/attention_monitor/
//	adb push build/attention_monitor /data/local/tmp/ && adb shell /data/local/tmp/attention_monitor
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/hardware/display"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

func main() {
	ctx := context.Background()

	driver, err := kernelbinder.Open(ctx, binder.WithMapSize(128*1024))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open binder: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close(ctx)

	transport, err := versionaware.NewTransport(ctx, driver, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "version-aware transport: %v\n", err)
		os.Exit(1)
	}

	sm := servicemanager.New(transport)

	fmt.Println("=== User Presence Monitor ===")
	fmt.Println()

	// PowerManager — primary source for user presence detection
	pm, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get power service: %v\n", err)
		os.Exit(1)
	}

	interactive, err := pm.IsInteractive(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsInteractive: %v\n", err)
	} else {
		fmt.Printf("Screen interactive:    %v\n", interactive)
	}

	idleMode, err := pm.IsDeviceIdleMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsDeviceIdleMode: %v\n", err)
	} else {
		fmt.Printf("Device idle (doze):    %v\n", idleMode)
	}

	lightIdle, err := pm.IsLightDeviceIdleMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsLightDeviceIdleMode: %v\n", err)
	} else {
		fmt.Printf("Light idle mode:       %v\n", lightIdle)
	}

	powerSave, err := pm.IsPowerSaveMode(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsPowerSaveMode: %v\n", err)
	} else {
		fmt.Printf("Battery saver:         %v\n", powerSave)
	}

	lowPowerStandby, err := pm.IsLowPowerStandbyEnabled(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IsLowPowerStandbyEnabled: %v\n", err)
	} else {
		fmt.Printf("Low-power standby:     %v\n", lowPowerStandby)
	}

	// DisplayManager — screen brightness as a presence signal
	fmt.Println()
	dmSvc, err := sm.GetService(ctx, servicemanager.DisplayService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get display service: %v\n", err)
	} else {
		dm := display.NewDisplayManagerProxy(dmSvc)

		ids, err := dm.GetDisplayIds(ctx, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GetDisplayIds: %v\n", err)
		} else {
			for _, id := range ids {
				brightness, err := dm.GetBrightness(ctx, id)
				if err != nil {
					fmt.Printf("Display %d brightness: error (%v)\n", id, err)
				} else {
					fmt.Printf("Display %d brightness: %.2f\n", id, brightness)
				}

				dispInteractive, err := pm.IsDisplayInteractive(ctx, id)
				if err != nil {
					fmt.Printf("Display %d interactive: error (%v)\n", id, err)
				} else {
					fmt.Printf("Display %d interactive: %v\n", id, dispInteractive)
				}
			}
		}
	}

	// Summary
	fmt.Println()
	if interactive && !idleMode {
		fmt.Println("User presence: LIKELY (screen on, device active)")
	} else if interactive {
		fmt.Println("User presence: UNCERTAIN (screen on but device idle)")
	} else {
		fmt.Println("User presence: UNLIKELY (screen off)")
	}
}
