// Collect GPS, battery, and device diagnostics for vehicle telematics.
//
// Aggregates data from multiple services: LocationManager (GPS),
// BatteryPropertiesRegistrar (battery), PowerManager (power state),
// and HardwarePropertiesManager (temperatures).
//
// Build:
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/vehicle_telematics ./examples/vehicle_telematics/
//	adb push build/vehicle_telematics /data/local/tmp/ && adb shell /data/local/tmp/vehicle_telematics
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AndroidGoLab/binder/android/location"
	genOs "github.com/AndroidGoLab/binder/android/os"
	"github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
)

const (
	batteryPropertyCapacity   = 4
	batteryPropertyCurrentNow = 2
	batteryPropertyStatus     = 6
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

	fmt.Println("=== Vehicle Telematics Report ===")
	fmt.Println()

	// GPS / Location
	fmt.Println("--- GPS ---")
	loc, err := location.GetLocationManager(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  LocationManager: %v\n", err)
	} else {
		providers, err := loc.GetAllProviders(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GetAllProviders: %v\n", err)
		} else {
			fmt.Printf("  Location providers: %v\n", providers)
		}

		enabled, err := loc.IsLocationEnabledForUser(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  IsLocationEnabled: %v\n", err)
		} else {
			fmt.Printf("  Location enabled:   %v\n", enabled)
		}

		gnssYear, err := loc.GetGnssYearOfHardware(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GNSS year: %v\n", err)
		} else {
			fmt.Printf("  GNSS hardware year: %d\n", gnssYear)
		}

		gnssModel, err := loc.GetGnssHardwareModelName(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  GNSS model: %v\n", err)
		} else {
			fmt.Printf("  GNSS model:         %s\n", gnssModel)
		}
	}
	fmt.Println()

	// Battery
	fmt.Println("--- Battery ---")
	battery, err := genOs.GetBatteryPropertiesRegistrar(ctx, sm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  BatteryProperties: %v\n", err)
	} else {
		cap, err := battery.GetProperty(ctx, batteryPropertyCapacity, genOs.BatteryProperty{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Battery level: %v\n", err)
		} else {
			fmt.Printf("  Battery level:      %d%%\n", cap)
		}

		current, err := battery.GetProperty(ctx, batteryPropertyCurrentNow, genOs.BatteryProperty{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Current draw: %v\n", err)
		} else {
			fmt.Printf("  Current draw:       %d uA\n", current)
		}

		status, err := battery.GetProperty(ctx, batteryPropertyStatus, genOs.BatteryProperty{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Battery status: %v\n", err)
		} else {
			names := map[int64]string{
				1: "Unknown", 2: "Charging", 3: "Discharging",
				4: "Not charging", 5: "Full",
			}
			name := names[int64(status)]
			if name == "" {
				name = fmt.Sprintf("(%d)", status)
			}
			fmt.Printf("  Battery status:     %s\n", name)
		}
	}
	fmt.Println()

	// Power state
	fmt.Println("--- Device State ---")
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
			fmt.Printf("  Device idle:        %v\n", idle)
		}
	}

	// Thermal
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
	}

	fmt.Println()
	fmt.Println("=== End Telematics ===")
}
