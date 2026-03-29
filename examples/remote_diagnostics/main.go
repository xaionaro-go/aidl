// Collect comprehensive device state for remote diagnostics.
//
// Queries multiple services: PowerManager (power state), DisplayManager
// (displays), HardwarePropertiesManager (temperatures), LocationManager
// (providers), and BatteryPropertiesRegistrar (battery).
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/remote_diagnostics ./examples/remote_diagnostics/
//	adb push build/remote_diagnostics /data/local/tmp/ && adb shell /data/local/tmp/remote_diagnostics
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/hardware/display"
	"github.com/AndroidGoLab/binder/android/location"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const batteryPropertyCapacity = 4

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

	fmt.Println("=== Remote Diagnostics Report ===")
	fmt.Println()

	// Power state
	fmt.Println("--- Power ---")
	power, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  PowerManager unavailable: %v\n", err)
	} else {
		interactive, err := power.IsInteractive(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsInteractive: %v\n", err)
		} else {
			fmt.Printf("  Screen on:            %v\n", interactive)
		}

		powerSave, err := power.IsPowerSaveMode(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsPowerSaveMode: %v\n", err)
		} else {
			fmt.Printf("  Power save mode:      %v\n", powerSave)
		}

		idle, err := power.IsDeviceIdleMode(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsDeviceIdleMode: %v\n", err)
		} else {
			fmt.Printf("  Device idle (Doze):   %v\n", idle)
		}
	}
	fmt.Println()

	// Battery
	fmt.Println("--- Battery ---")
	battery, err := genOs.GetBatteryPropertiesRegistrar(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  BatteryProperties unavailable: %v\n", err)
	} else {
		cap, err := battery.GetProperty(ctx, batteryPropertyCapacity, genOs.BatteryProperty{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Battery level: %v\n", err)
		} else {
			fmt.Printf("  Battery level:        %d%%\n", cap)
		}
	}
	fmt.Println()

	// Display
	fmt.Println("--- Display ---")
	dispSvc, err := sm.GetService(ctx, servicemanager.DisplayService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  DisplayManager unavailable: %v\n", err)
	} else {
		dm := display.NewDisplayManagerProxy(dispSvc)
		ids, err := dm.GetDisplayIds(ctx, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetDisplayIds: %v\n", err)
		} else {
			fmt.Printf("  Displays:             %d\n", len(ids))
			for _, id := range ids {
				b, err := dm.GetBrightness(ctx, id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Display %d brightness: %v\n", id, err)
				} else {
					fmt.Printf("  Display %d brightness: %.2f\n", id, b)
				}
			}
		}
	}
	fmt.Println()

	// Temperature
	fmt.Println("--- Thermal ---")
	hwProps, err := genOs.GetHardwarePropertiesManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  HardwarePropertiesManager unavailable: %v\n", err)
	} else {
		// type=0 (DEVICE_TEMPERATURE_CPU), source=0 (TEMPERATURE_CURRENT)
		temps, err := hwProps.GetDeviceTemperatures(ctx, 0, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  CPU temperatures: %v\n", err)
		} else {
			fmt.Printf("  CPU temperatures:     %v\n", temps)
		}
	}
	fmt.Println()

	// Location providers
	fmt.Println("--- Location ---")
	loc, err := location.GetLocationManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  LocationManager unavailable: %v\n", err)
	} else {
		providers, err := loc.GetAllProviders(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetAllProviders: %v\n", err)
		} else {
			fmt.Printf("  Location providers:   %v\n", providers)
		}

		enabled, err := loc.IsLocationEnabledForUser(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsLocationEnabled: %v\n", err)
		} else {
			fmt.Printf("  Location enabled:     %v\n", enabled)
		}
	}

	fmt.Println()
	fmt.Println("=== End Diagnostics ===")
}
