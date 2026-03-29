// Headless device orchestration: query power, display, and process state.
//
// Designed for devices running without a UI (kiosks, IoT gateways).
// Queries power state, display configuration, and running processes
// to provide a control plane overview.
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/headless_controller ./examples/headless_controller/
//	adb push build/headless_controller /data/local/tmp/ && adb shell /data/local/tmp/headless_controller
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

	fmt.Println("=== Headless Controller ===")
	fmt.Println()

	// Power state
	fmt.Println("--- Power State ---")
	power, err := genOs.GetPowerManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  PowerManager: %v\n", err)
	} else {
		interactive, err := power.IsInteractive(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsInteractive: %v\n", err)
		} else {
			fmt.Printf("  Screen on:          %v\n", interactive)
		}

		idle, err := power.IsDeviceIdleMode(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsDeviceIdleMode: %v\n", err)
		} else {
			fmt.Printf("  Device idle (Doze): %v\n", idle)
		}

		powerSave, err := power.IsPowerSaveMode(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsPowerSaveMode: %v\n", err)
		} else {
			fmt.Printf("  Power save mode:    %v\n", powerSave)
		}

		lowPower, err := power.IsLowPowerStandbyEnabled(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsLowPowerStandbyEnabled: %v\n", err)
		} else {
			fmt.Printf("  Low power standby:  %v\n", lowPower)
		}
	}
	fmt.Println()

	// Display state
	fmt.Println("--- Display State ---")
	dispSvc, err := sm.GetService(ctx, servicemanager.DisplayService)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  DisplayManager: %v\n", err)
	} else {
		dm := display.NewDisplayManagerProxy(dispSvc)
		ids, err := dm.GetDisplayIds(ctx, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetDisplayIds: %v\n", err)
		} else {
			fmt.Printf("  Connected displays: %d\n", len(ids))
			for _, id := range ids {
				b, err := dm.GetBrightness(ctx, id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Display %d: %v\n", id, err)
				} else {
					fmt.Printf("  Display %d:          brightness=%.2f\n", id, b)
				}
			}
		}

		stableSize, err := dm.GetStableDisplaySize(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetStableDisplaySize: %v\n", err)
		} else {
			fmt.Printf("  Stable display size: %dx%d\n", stableSize.X, stableSize.Y)
		}
	}
	fmt.Println()

	// Hardware temperatures
	fmt.Println("--- Thermal State ---")
	hwProps, err := genOs.GetHardwarePropertiesManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  HardwarePropertiesManager: %v\n", err)
	} else {
		temps, err := hwProps.GetDeviceTemperatures(ctx, 0, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  CPU temperatures: %v\n", err)
		} else {
			fmt.Printf("  CPU temperatures:   %v\n", temps)
		}

		fans, err := hwProps.GetFanSpeeds(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Fan speeds: %v\n", err)
		} else {
			if len(fans) == 0 {
				fmt.Printf("  Fan speeds:         (no fans)\n")
			} else {
				fmt.Printf("  Fan speeds:         %v\n", fans)
			}
		}
	}

	fmt.Println()
	fmt.Println("=== End Headless Controller ===")
}
